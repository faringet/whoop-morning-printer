package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/fieldnote"
	"log/slog"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/art"
	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/render"
	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/storage"
)

type Config struct {
	UserID int64

	Timezone string

	Interval  time.Duration
	PollLimit int

	ProcessWakeReceipt bool
	ProcessFinalReport bool

	EnsureFinalReportJobs bool

	FinalReportRequireAdvice bool
	FallbackAfterDeadline    bool

	ReceiptWidth         int
	ReceiptLineSeparator string

	ArtEnabled  bool
	ArtMode     string
	ArtMaxLines int
}

type Worker struct {
	log *slog.Logger
	cfg Config

	store              storage.Store
	artSelector        *art.Selector
	fieldNoteGenerator fieldnote.Generator

	now func() time.Time
}

func New(log *slog.Logger, cfg Config, store storage.Store, artSelector *art.Selector, fieldNoteGenerator fieldnote.Generator) (*Worker, error) {
	if log == nil {
		log = slog.Default()
	}
	if store == nil {
		return nil, errors.New("receiptworker: store is nil")
	}

	cfg.Timezone = strings.TrimSpace(cfg.Timezone)
	if cfg.Timezone == "" {
		cfg.Timezone = "Europe/Moscow"
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return nil, fmt.Errorf("receiptworker: invalid timezone: %w", err)
	}

	if cfg.UserID <= 0 {
		return nil, errors.New("receiptworker: user_id must be > 0")
	}

	if cfg.Interval <= 0 {
		cfg.Interval = 15 * time.Second
	}
	if cfg.PollLimit <= 0 {
		cfg.PollLimit = 20
	}

	if cfg.ReceiptWidth <= 0 {
		cfg.ReceiptWidth = 42
	}

	cfg.ReceiptLineSeparator = strings.TrimSpace(cfg.ReceiptLineSeparator)
	if cfg.ReceiptLineSeparator == "" {
		cfg.ReceiptLineSeparator = "-"
	}

	cfg.ArtMode = strings.ToLower(strings.TrimSpace(cfg.ArtMode))
	if cfg.ArtMode == "" {
		cfg.ArtMode = art.ModeDeterministic
	}

	if cfg.ArtMaxLines <= 0 {
		cfg.ArtMaxLines = 8
	}

	return &Worker{
		log: log.With(
			slog.String("layer", "worker"),
			slog.String("module", "receiptworker.worker"),
		),
		cfg:                cfg,
		store:              store,
		artSelector:        artSelector,
		fieldNoteGenerator: fieldNoteGenerator,
		now:                time.Now,
	}, nil
}

func (w *Worker) RunOnce(ctx context.Context) error {
	if w == nil {
		return errors.New("receiptworker: worker is nil")
	}

	w.log.Info("run once started",
		slog.Int64("user_id", w.cfg.UserID),
		slog.String("timezone", w.cfg.Timezone),
		slog.Int("poll_limit", w.cfg.PollLimit),
		slog.Bool("process_wake_receipt", w.cfg.ProcessWakeReceipt),
		slog.Bool("process_final_report", w.cfg.ProcessFinalReport),
		slog.Bool("ensure_final_report_jobs", w.cfg.EnsureFinalReportJobs),
		slog.Bool("final_report_require_advice", w.cfg.FinalReportRequireAdvice),
		slog.Bool("fallback_after_deadline", w.cfg.FallbackAfterDeadline),
		slog.Int("receipt_width", w.cfg.ReceiptWidth),
		slog.Bool("art_enabled", w.cfg.ArtEnabled),
		slog.String("art_mode", w.cfg.ArtMode),
	)

	return w.tick(ctx)
}

func (w *Worker) RunInterval(ctx context.Context) error {
	if w == nil {
		return errors.New("receiptworker: worker is nil")
	}

	w.log.Info("interval worker started",
		slog.Int64("user_id", w.cfg.UserID),
		slog.String("timezone", w.cfg.Timezone),
		slog.Duration("interval", w.cfg.Interval),
		slog.Int("poll_limit", w.cfg.PollLimit),
		slog.Bool("process_wake_receipt", w.cfg.ProcessWakeReceipt),
		slog.Bool("process_final_report", w.cfg.ProcessFinalReport),
		slog.Bool("ensure_final_report_jobs", w.cfg.EnsureFinalReportJobs),
		slog.Bool("final_report_require_advice", w.cfg.FinalReportRequireAdvice),
		slog.Bool("fallback_after_deadline", w.cfg.FallbackAfterDeadline),
		slog.Int("receipt_width", w.cfg.ReceiptWidth),
		slog.Bool("art_enabled", w.cfg.ArtEnabled),
		slog.String("art_mode", w.cfg.ArtMode),
	)

	if err := w.tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
		w.log.Warn("initial tick failed", slog.Any("err", err))
	}

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("interval worker stopped", slog.Any("reason", ctx.Err()))
			return ctx.Err()

		case <-ticker.C:
			if err := w.tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
				w.log.Warn("tick failed", slog.Any("err", err))
			}
		}
	}
}

func (w *Worker) tick(ctx context.Context) error {
	now := w.now().UTC()

	if w.cfg.EnsureFinalReportJobs {
		created, err := w.store.EnsureFinalReportJobs(ctx, storage.EnsureFinalReportJobsInput{
			UserID: w.cfg.UserID,
			Now:    now,
			Limit:  w.cfg.PollLimit,
		})
		if err != nil {
			return fmt.Errorf("receiptworker: ensure final report jobs: %w", err)
		}

		if created > 0 {
			w.log.Info("final report jobs ensured",
				slog.Int("created", created),
				slog.Int64("user_id", w.cfg.UserID),
			)
		}
	}

	if w.cfg.ProcessWakeReceipt {
		if err := w.processWakeReceipts(ctx, now); err != nil {
			return err
		}
	}

	if w.cfg.ProcessFinalReport {
		if err := w.processFinalReports(ctx, now); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) processWakeReceipts(ctx context.Context, now time.Time) error {
	tasks, err := w.store.GetPendingWakeReceiptTasks(ctx, storage.GetPendingTasksInput{
		UserID: w.cfg.UserID,
		Now:    now,
		Limit:  w.cfg.PollLimit,
	})
	if err != nil {
		return fmt.Errorf("receiptworker: get pending wake receipt tasks: %w", err)
	}

	if len(tasks) == 0 {
		w.log.Debug("no pending wake receipt tasks",
			slog.Int64("user_id", w.cfg.UserID),
		)
		return nil
	}

	for _, task := range tasks {
		if err := w.processWakeReceiptTask(ctx, task); err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}

			w.log.Warn("process wake receipt task failed",
				slog.Int64("print_job_id", task.PrintJob.ID),
				slog.Int64("wake_plan_id", task.WakePlan.ID),
				slog.Any("err", err),
			)
		}
	}

	return nil
}

func (w *Worker) processWakeReceiptTask(ctx context.Context, task storage.WakeReceiptTask) error {
	selection := art.Selection{}

	if w.artSelector != nil {
		selection = w.artSelector.Pick(art.PickInput{
			Enabled:  w.cfg.ArtEnabled,
			Mode:     w.cfg.ArtMode,
			UserID:   task.WakePlan.UserID,
			Date:     task.WakePlan.Date,
			Width:    w.cfg.ReceiptWidth,
			MaxLines: w.cfg.ArtMaxLines,
			Salt:     fmt.Sprintf("wake_plan:%d:wake_receipt", task.WakePlan.ID),
		})
	}

	fieldNoteText := ""

	if w.fieldNoteGenerator != nil {
		startedAt := time.Now()

		result, err := w.fieldNoteGenerator.Generate(ctx, fieldnote.Input{
			UserID:     task.WakePlan.UserID,
			WakePlanID: task.WakePlan.ID,
			Date:       task.WakePlan.Date,
		})
		if err != nil {
			w.log.Warn("field note generation failed",
				slog.Int64("wake_plan_id", task.WakePlan.ID),
				slog.Duration("duration", time.Since(startedAt)),
				slog.Any("err", err),
			)
		} else {
			fieldNoteText = result.Text

			w.log.Info("field note generated",
				slog.Int64("wake_plan_id", task.WakePlan.ID),
				slog.String("source", string(result.Source)),
				slog.Duration("duration", time.Since(startedAt)),
			)
		}
	}

	payloadText, err := render.RenderWakeReceipt(render.WakeReceiptInput{
		Task: task,

		Timezone: w.cfg.Timezone,

		Width:         w.cfg.ReceiptWidth,
		LineSeparator: w.cfg.ReceiptLineSeparator,

		ArtText: selection.Text,

		FieldNote: fieldNoteText,
	})
	if err != nil {
		_, markErr := w.store.MarkPrintJobFailed(ctx, storage.MarkPrintJobFailedInput{
			PrintJobID:   task.PrintJob.ID,
			ErrorMessage: err.Error(),
		})
		if markErr != nil {
			return fmt.Errorf("render wake receipt failed: %w; mark failed: %w", err, markErr)
		}

		return fmt.Errorf("render wake receipt failed: %w", err)
	}

	job, err := w.store.MarkPrintJobReady(ctx, storage.MarkPrintJobReadyInput{
		PrintJobID: task.PrintJob.ID,

		PayloadType: string(storage.PayloadTypeTextPlain),
		PayloadText: payloadText,

		NotBefore: task.WakePlan.WakeAt,
	})
	if err != nil {
		return fmt.Errorf("mark wake receipt ready: %w", err)
	}

	w.log.Info("wake receipt prepared",
		slog.Int64("print_job_id", job.ID),
		slog.Int64("wake_plan_id", task.WakePlan.ID),
		slog.Time("not_before", job.NotBefore),
		slog.String("art", selection.Name),
	)

	return nil
}

func (w *Worker) processFinalReports(ctx context.Context, now time.Time) error {
	tasks, err := w.store.GetPendingFinalReportTasks(ctx, storage.GetPendingTasksInput{
		UserID: w.cfg.UserID,
		Now:    now,
		Limit:  w.cfg.PollLimit,
	})
	if err != nil {
		return fmt.Errorf("receiptworker: get pending final report tasks: %w", err)
	}

	if len(tasks) == 0 {
		w.log.Debug("no pending final report tasks",
			slog.Int64("user_id", w.cfg.UserID),
		)
		return nil
	}

	for _, task := range tasks {
		if err := w.processFinalReportTask(ctx, task, now); err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}

			w.log.Warn("process final report task failed",
				slog.Int64("print_job_id", task.PrintJob.ID),
				slog.Int64("wake_plan_id", task.WakePlan.ID),
				slog.Any("err", err),
			)
		}
	}

	return nil
}

func (w *Worker) processFinalReportTask(ctx context.Context, task storage.FinalReportTask, now time.Time) error {
	if task.HasReadyData(w.cfg.FinalReportRequireAdvice) {
		return w.prepareFinalReport(ctx, task, now)
	}

	if w.cfg.FallbackAfterDeadline && task.IsPastDeadline(now) {
		return w.prepareFallbackReport(ctx, task, now)
	}

	w.log.Debug("final report data is not ready yet",
		slog.Int64("print_job_id", task.PrintJob.ID),
		slog.Int64("wake_plan_id", task.WakePlan.ID),
		slog.Bool("snapshot_ready", task.Snapshot != nil && task.Snapshot.IsReady()),
		slog.Bool("advice_ready", task.Advice != nil && task.Advice.IsReady()),
		slog.Bool("require_advice", w.cfg.FinalReportRequireAdvice),
		slog.Time("final_deadline_at", task.WakePlan.FinalDeadlineAt),
	)

	return nil
}

func (w *Worker) prepareFinalReport(ctx context.Context, task storage.FinalReportTask, now time.Time) error {
	payloadText, err := render.RenderFinalReport(render.FinalReportInput{
		Task: task,

		Timezone: w.cfg.Timezone,

		Width:         w.cfg.ReceiptWidth,
		LineSeparator: w.cfg.ReceiptLineSeparator,
	})
	if err != nil {
		_, markErr := w.store.MarkPrintJobFailed(ctx, storage.MarkPrintJobFailedInput{
			PrintJobID:   task.PrintJob.ID,
			ErrorMessage: err.Error(),
		})
		if markErr != nil {
			return fmt.Errorf("render final report failed: %w; mark failed: %w", err, markErr)
		}

		return fmt.Errorf("render final report failed: %w", err)
	}

	job, err := w.store.MarkPrintJobReady(ctx, storage.MarkPrintJobReadyInput{
		PrintJobID: task.PrintJob.ID,

		PayloadType: string(storage.PayloadTypeTextPlain),
		PayloadText: payloadText,

		NotBefore: now,
	})
	if err != nil {
		return fmt.Errorf("mark final report ready: %w", err)
	}

	w.log.Info("final report prepared",
		slog.Int64("print_job_id", job.ID),
		slog.Int64("wake_plan_id", task.WakePlan.ID),
		slog.Time("not_before", job.NotBefore),
		slog.Bool("has_advice", task.Advice != nil && task.Advice.IsReady()),
	)

	return nil
}

func (w *Worker) prepareFallbackReport(ctx context.Context, task storage.FinalReportTask, now time.Time) error {
	payloadText, err := render.RenderFallbackReport(render.FallbackReportInput{
		Task: task,

		Timezone: w.cfg.Timezone,

		Width:         w.cfg.ReceiptWidth,
		LineSeparator: w.cfg.ReceiptLineSeparator,
	})
	if err != nil {
		_, markErr := w.store.MarkPrintJobFailed(ctx, storage.MarkPrintJobFailedInput{
			PrintJobID:   task.PrintJob.ID,
			ErrorMessage: err.Error(),
		})
		if markErr != nil {
			return fmt.Errorf("render fallback report failed: %w; mark failed: %w", err, markErr)
		}

		return fmt.Errorf("render fallback report failed: %w", err)
	}

	job, err := w.store.MarkPrintJobReady(ctx, storage.MarkPrintJobReadyInput{
		PrintJobID: task.PrintJob.ID,

		PayloadType: string(storage.PayloadTypeTextPlain),
		PayloadText: payloadText,

		NotBefore: now,
	})
	if err != nil {
		return fmt.Errorf("mark fallback report ready: %w", err)
	}

	w.log.Info("fallback report prepared",
		slog.Int64("print_job_id", job.ID),
		slog.Int64("wake_plan_id", task.WakePlan.ID),
		slog.Time("not_before", job.NotBefore),
		slog.Bool("has_snapshot", task.Snapshot != nil),
		slog.Bool("has_advice", task.Advice != nil),
	)

	return nil
}
