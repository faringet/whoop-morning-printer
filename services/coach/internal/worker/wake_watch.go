package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/faringet/whoop-morning-printer/services/coach/internal/storage"
)

func (w *Worker) RunWakeWatch(ctx context.Context) error {
	if w == nil {
		return errors.New("coach wake_watch: worker is nil")
	}

	w.log.Info("wake_watch worker started",
		slog.Int64("user_id", w.cfg.UserID),
		slog.String("timezone", w.cfg.Timezone),
		slog.String("model", w.cfg.Model),
		slog.String("prompt_version", w.cfg.PromptVersion),
		slog.Duration("poll_interval", w.cfg.PollInterval),
		slog.Duration("warmup_before_wake", w.cfg.WarmupBeforeWake),
		slog.Duration("warmup_timeout", w.cfg.WarmupTimeout),
		slog.Duration("min_warmup_interval", w.cfg.MinWarmupInterval),
		slog.Int("active_wake_plan_limit", w.cfg.ActiveWakePlanLimit),
		slog.Bool("require_ready_snapshot", w.cfg.RequireReadySnapshot),
		slog.Bool("allow_partial_after_deadline", w.cfg.AllowPartialAfterDeadline),
	)

	if w.cfg.WarmupOnStart {
		if err := w.warmupModel(ctx, "wake_watch_start", true); err != nil {
			w.log.Warn("ollama warmup failed before wake_watch", slog.Any("err", err))
		}
	}

	if err := w.tickWakeWatch(ctx); err != nil && !errors.Is(err, context.Canceled) {
		w.log.Warn("initial wake_watch tick failed", slog.Any("err", err))
	}

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("wake_watch worker stopped", slog.Any("reason", ctx.Err()))
			return ctx.Err()

		case <-ticker.C:
			if err := w.tickWakeWatch(ctx); err != nil && !errors.Is(err, context.Canceled) {
				w.log.Warn("wake_watch tick failed", slog.Any("err", err))
			}
		}
	}
}

func (w *Worker) tickWakeWatch(ctx context.Context) error {
	now := w.now().UTC()

	wakePlans, err := w.store.GetActiveWakePlans(ctx, storage.GetActiveWakePlansInput{
		UserID: w.cfg.UserID,
		Now:    now,
		Limit:  w.cfg.ActiveWakePlanLimit,
	})
	if err != nil {
		return fmt.Errorf("coach wake_watch: get active wake plans: %w", err)
	}

	if len(wakePlans) == 0 {
		w.log.Debug("no active wake plans found",
			slog.Int64("user_id", w.cfg.UserID),
		)
		return nil
	}

	for _, wakePlan := range wakePlans {
		if err := w.processWakePlan(ctx, wakePlan, now); err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}

			w.log.Warn("process wake plan failed",
				slog.Int64("wake_plan_id", wakePlan.ID),
				slog.Int64("user_id", wakePlan.UserID),
				slog.Time("date", wakePlan.Date),
				slog.Time("wake_at", wakePlan.WakeAt),
				slog.Any("err", err),
			)
		}
	}

	return nil
}

func (w *Worker) processWakePlan(ctx context.Context, wakePlan storage.WakePlan, now time.Time) error {
	if wakePlan.ID <= 0 {
		return errors.New("coach wake_watch: wake_plan.id must be > 0")
	}

	existingAdvice, err := w.store.GetMorningAdvice(ctx, storage.GetMorningAdviceInput{
		UserID:        wakePlan.UserID,
		Date:          wakePlan.Date,
		PromptVersion: w.cfg.PromptVersion,
	})
	if err == nil {
		w.log.Debug("morning advice already exists for wake plan",
			slog.Int64("wake_plan_id", wakePlan.ID),
			slog.Int64("advice_id", existingAdvice.ID),
			slog.Time("date", wakePlan.Date),
			slog.String("prompt_version", w.cfg.PromptVersion),
		)
		return nil
	}
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("coach wake_watch: check existing advice: %w", err)
	}

	if shouldWarmupForWakePlan(now, wakePlan, w.cfg.WarmupBeforeWake) {
		if err := w.warmupModel(ctx, "before_wake", false); err != nil {
			w.log.Warn("ollama warmup failed near wake time",
				slog.Int64("wake_plan_id", wakePlan.ID),
				slog.Time("wake_at", wakePlan.WakeAt),
				slog.Any("err", err),
			)
		}
	}

	snapshot, err := w.store.GetSnapshotForWakePlan(ctx, storage.GetSnapshotForWakePlanInput{
		UserID:       wakePlan.UserID,
		WakePlanDate: wakePlan.Date,

		LookbackDays: w.cfg.SnapshotLookbackDays,

		RequireReadySnapshot:      w.cfg.RequireReadySnapshot,
		AllowPartialAfterDeadline: w.cfg.AllowPartialAfterDeadline,
		Deadline:                  wakePlan.FinalDeadlineAt,
	})
	if errors.Is(err, storage.ErrNotFound) {
		w.log.Debug("snapshot is not ready for wake plan yet",
			slog.Int64("wake_plan_id", wakePlan.ID),
			slog.Int64("user_id", wakePlan.UserID),
			slog.Time("date", wakePlan.Date),
			slog.Time("wake_at", wakePlan.WakeAt),
			slog.Time("final_deadline_at", wakePlan.FinalDeadlineAt),
			slog.Bool("require_ready_snapshot", w.cfg.RequireReadySnapshot),
			slog.Bool("allow_partial_after_deadline", w.cfg.AllowPartialAfterDeadline),
		)
		return nil
	}
	if err != nil {
		return fmt.Errorf("coach wake_watch: get snapshot for wake plan: %w", err)
	}

	w.log.Info("snapshot found for wake plan, generating advice",
		slog.Int64("wake_plan_id", wakePlan.ID),
		slog.Int64("snapshot_id", snapshot.ID),
		slog.Int64("user_id", snapshot.UserID),
		slog.Time("date", snapshot.Date),
		slog.String("data_state", string(snapshot.DataState)),
	)

	wakeAt := wakePlan.WakeAt

	return w.processSnapshot(ctx, snapshot, &wakeAt)
}

func shouldWarmupForWakePlan(now time.Time, wakePlan storage.WakePlan, warmupBeforeWake time.Duration) bool {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if warmupBeforeWake <= 0 {
		return false
	}

	warmupAt := wakePlan.WarmupAt(warmupBeforeWake)

	if now.Before(warmupAt) {
		return false
	}

	if !wakePlan.FinalDeadlineAt.IsZero() && now.After(wakePlan.FinalDeadlineAt) {
		return false
	}

	return true
}
