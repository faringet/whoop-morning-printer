package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	coachadvisor "github.com/faringet/whoop-morning-printer/services/coach/internal/advisor"
	"github.com/faringet/whoop-morning-printer/services/coach/internal/storage"
)

type WarmupClient interface {
	Warmup(ctx context.Context, model string) error
}

type Config struct {
	UserID int64

	Timezone string

	Model         string
	PromptVersion string

	Interval     time.Duration
	PollInterval time.Duration

	SnapshotLookbackDays int

	RequireReadySnapshot      bool
	AllowPartialAfterDeadline bool

	WarmupOnStart       bool
	WarmupBeforeWake    time.Duration
	WarmupTimeout       time.Duration
	MinWarmupInterval   time.Duration
	ActiveWakePlanLimit int
}

type Worker struct {
	log *slog.Logger
	cfg Config

	store   storage.Store
	advisor *coachadvisor.Advisor
	warmup  WarmupClient

	lastWarmupAt time.Time
	now          func() time.Time
}

func New(log *slog.Logger, cfg Config, store storage.Store, advisor *coachadvisor.Advisor, warmup WarmupClient) (*Worker, error) {
	if log == nil {
		log = slog.Default()
	}
	if store == nil {
		return nil, errors.New("coach worker: store is nil")
	}
	if advisor == nil {
		return nil, errors.New("coach worker: advisor is nil")
	}

	cfg.Timezone = strings.TrimSpace(cfg.Timezone)
	if cfg.Timezone == "" {
		cfg.Timezone = "Europe/Moscow"
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return nil, fmt.Errorf("coach worker: invalid timezone: %w", err)
	}

	if cfg.UserID <= 0 {
		return nil, errors.New("coach worker: user_id must be > 0")
	}

	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.Model == "" {
		return nil, errors.New("coach worker: model is required")
	}

	cfg.PromptVersion = strings.TrimSpace(cfg.PromptVersion)
	if cfg.PromptVersion == "" {
		return nil, errors.New("coach worker: prompt_version is required")
	}

	if cfg.Interval <= 0 {
		cfg.Interval = 10 * time.Minute
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 15 * time.Second
	}
	if cfg.SnapshotLookbackDays <= 0 {
		cfg.SnapshotLookbackDays = 3
	}
	if cfg.WarmupBeforeWake <= 0 {
		cfg.WarmupBeforeWake = 15 * time.Minute
	}
	if cfg.WarmupTimeout <= 0 {
		cfg.WarmupTimeout = 2 * time.Minute
	}
	if cfg.MinWarmupInterval <= 0 {
		cfg.MinWarmupInterval = 30 * time.Minute
	}
	if cfg.ActiveWakePlanLimit <= 0 {
		cfg.ActiveWakePlanLimit = 10
	}

	return &Worker{
		log: log.With(
			slog.String("layer", "worker"),
			slog.String("module", "coach.worker"),
		),
		cfg:     cfg,
		store:   store,
		advisor: advisor,
		warmup:  warmup,
		now:     time.Now,
	}, nil
}

func (w *Worker) RunOnce(ctx context.Context) error {
	if w == nil {
		return errors.New("coach worker: worker is nil")
	}

	w.log.Info("run once started",
		slog.Int64("user_id", w.cfg.UserID),
		slog.String("timezone", w.cfg.Timezone),
		slog.String("model", w.cfg.Model),
		slog.String("prompt_version", w.cfg.PromptVersion),
		slog.Int("snapshot_lookback_days", w.cfg.SnapshotLookbackDays),
		slog.Bool("require_ready_snapshot", w.cfg.RequireReadySnapshot),
		slog.Bool("allow_partial_after_deadline", w.cfg.AllowPartialAfterDeadline),
	)

	if w.cfg.WarmupOnStart {
		if err := w.warmupModel(ctx, "run_once_start", true); err != nil {
			w.log.Warn("ollama warmup failed before run once", slog.Any("err", err))
		}
	}

	return w.processLatestSnapshot(ctx)
}

func (w *Worker) RunInterval(ctx context.Context) error {
	if w == nil {
		return errors.New("coach worker: worker is nil")
	}

	w.log.Info("interval worker started",
		slog.Int64("user_id", w.cfg.UserID),
		slog.String("timezone", w.cfg.Timezone),
		slog.String("model", w.cfg.Model),
		slog.String("prompt_version", w.cfg.PromptVersion),
		slog.Duration("interval", w.cfg.Interval),
		slog.Int("snapshot_lookback_days", w.cfg.SnapshotLookbackDays),
		slog.Bool("require_ready_snapshot", w.cfg.RequireReadySnapshot),
		slog.Bool("allow_partial_after_deadline", w.cfg.AllowPartialAfterDeadline),
	)

	if w.cfg.WarmupOnStart {
		if err := w.warmupModel(ctx, "interval_start", true); err != nil {
			w.log.Warn("ollama warmup failed before interval worker", slog.Any("err", err))
		}
	}

	if err := w.processLatestSnapshot(ctx); err != nil && !errors.Is(err, context.Canceled) {
		w.log.Warn("initial coach tick failed", slog.Any("err", err))
	}

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("interval worker stopped", slog.Any("reason", ctx.Err()))
			return ctx.Err()

		case <-ticker.C:
			if err := w.processLatestSnapshot(ctx); err != nil && !errors.Is(err, context.Canceled) {
				w.log.Warn("coach tick failed", slog.Any("err", err))
			}
		}
	}
}

func (w *Worker) processLatestSnapshot(ctx context.Context) error {
	snapshot, err := w.store.GetLatestSnapshot(ctx, storage.GetLatestSnapshotInput{
		UserID: w.cfg.UserID,

		LookbackDays: w.cfg.SnapshotLookbackDays,

		RequireReadySnapshot:      w.cfg.RequireReadySnapshot,
		AllowPartialAfterDeadline: w.cfg.AllowPartialAfterDeadline,
	})
	if errors.Is(err, storage.ErrNotFound) {
		w.log.Info("no suitable snapshot found",
			slog.Int64("user_id", w.cfg.UserID),
			slog.Int("lookback_days", w.cfg.SnapshotLookbackDays),
			slog.Bool("require_ready_snapshot", w.cfg.RequireReadySnapshot),
		)
		return nil
	}
	if err != nil {
		return fmt.Errorf("coach worker: get latest snapshot: %w", err)
	}

	return w.processSnapshot(ctx, snapshot, nil)
}

func (w *Worker) processSnapshot(ctx context.Context, snapshot storage.DailyHealthSnapshot, wakeAt *time.Time) error {
	if snapshot.ID <= 0 {
		return errors.New("coach worker: snapshot.id must be > 0")
	}

	_, err := w.store.GetMorningAdvice(ctx, storage.GetMorningAdviceInput{
		UserID:        snapshot.UserID,
		Date:          snapshot.Date,
		PromptVersion: w.cfg.PromptVersion,
	})
	if err == nil {
		w.log.Info("morning advice already exists, skip generation",
			slog.Int64("user_id", snapshot.UserID),
			slog.Int64("snapshot_id", snapshot.ID),
			slog.Time("date", snapshot.Date),
			slog.String("prompt_version", w.cfg.PromptVersion),
		)
		return nil
	}
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("coach worker: check existing advice: %w", err)
	}

	advice, err := w.advisor.Build(ctx, coachadvisor.BuildInput{
		Snapshot: toAdvisorSnapshot(snapshot),
		Timezone: w.cfg.Timezone,
		WakeAt:   wakeAt,
	})
	if err != nil {
		return fmt.Errorf("coach worker: build advice: %w", err)
	}

	saved, err := w.store.UpsertMorningAdvice(ctx, storage.UpsertMorningAdviceInput{
		UserID: snapshot.UserID,

		Date:       advice.Date,
		SnapshotID: advice.SnapshotID,

		Model:         advice.Model,
		PromptVersion: advice.PromptVersion,

		DayType:    string(advice.DayType),
		MainSignal: advice.MainSignal,
		AdviceText: advice.AdviceText,
		Motto:      advice.Motto,

		PayloadJSON: advice.PayloadJSON,
		GeneratedAt: advice.GeneratedAt,
	})
	if err != nil {
		return fmt.Errorf("coach worker: save advice: %w", err)
	}

	w.log.Info("morning advice generated",
		slog.Int64("advice_id", saved.ID),
		slog.Int64("user_id", saved.UserID),
		slog.Int64("snapshot_id", saved.SnapshotID),
		slog.Time("date", saved.Date),
		slog.String("day_type", saved.DayType),
		slog.String("main_signal", saved.MainSignal),
		slog.String("model", saved.Model),
		slog.String("prompt_version", saved.PromptVersion),
	)

	return nil
}

func (w *Worker) warmupModel(ctx context.Context, reason string, force bool) error {
	if w == nil {
		return errors.New("coach worker: worker is nil")
	}
	if w.warmup == nil {
		return nil
	}

	now := w.now().UTC()

	if !force && !w.lastWarmupAt.IsZero() {
		if since := now.Sub(w.lastWarmupAt); since < w.cfg.MinWarmupInterval {
			w.log.Debug("skip ollama warmup: min interval not reached",
				slog.String("reason", reason),
				slog.Duration("since_last_warmup", since),
				slog.Duration("min_warmup_interval", w.cfg.MinWarmupInterval),
			)
			return nil
		}
	}

	timeout := w.cfg.WarmupTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	warmupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	w.log.Info("ollama warmup started",
		slog.String("reason", reason),
		slog.String("model", w.cfg.Model),
		slog.Duration("timeout", timeout),
	)

	if err := w.warmup.Warmup(warmupCtx, w.cfg.Model); err != nil {
		return err
	}

	w.lastWarmupAt = w.now().UTC()

	w.log.Info("ollama warmup completed",
		slog.String("reason", reason),
		slog.String("model", w.cfg.Model),
	)

	return nil
}

func toAdvisorSnapshot(snapshot storage.DailyHealthSnapshot) coachadvisor.Snapshot {
	return coachadvisor.Snapshot{
		ID:     snapshot.ID,
		UserID: snapshot.UserID,

		Date:      snapshot.Date,
		DataState: string(snapshot.DataState),

		SleepScore:    snapshot.SleepScore,
		RecoveryScore: snapshot.RecoveryScore,
		DayStrain:     snapshot.DayStrain,

		SleepMinutes:       snapshot.SleepMinutes,
		SleepNeededMinutes: snapshot.SleepNeededMinutes,
		SleepVsNeedPct:     snapshot.SleepVsNeedPct,

		AwakeMinutes:       snapshot.AwakeMinutes,
		LightSleepMinutes:  snapshot.LightSleepMinutes,
		DeepSleepMinutes:   snapshot.DeepSleepMinutes,
		REMSleepMinutes:    snapshot.REMSleepMinutes,
		RestorativeMinutes: snapshot.RestorativeMinutes,

		SleepEfficiencyPct:  snapshot.SleepEfficiencyPct,
		SleepConsistencyPct: snapshot.SleepConsistencyPct,

		RespiratoryRate:     snapshot.RespiratoryRate,
		HRVRMSSDMS:          snapshot.HRVRMSSDMS,
		RestingHeartRateBPM: snapshot.RestingHeartRateBPM,
		SpO2Pct:             snapshot.SpO2Pct,
		SkinTempCelsius:     snapshot.SkinTempCelsius,

		SourceUpdatedAt: snapshot.SourceUpdatedAt,
	}
}
