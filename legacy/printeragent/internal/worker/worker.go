package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/logger"
	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/output"
	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/storage"
)

type Config struct {
	UserID int64

	Interval  time.Duration
	PollLimit int

	WorkerID string
	ClaimTTL time.Duration

	PrintDelay time.Duration
}

type Worker struct {
	log *logger.Logger
	cfg Config

	store   storage.Store
	printer output.Printer

	now func() time.Time
}

func New(
	log *logger.Logger,
	cfg Config,
	store storage.Store,
	printer output.Printer,
) (*Worker, error) {
	if store == nil {
		return nil, errors.New("printeragent legacy worker: store is nil")
	}
	if printer == nil {
		return nil, errors.New("printeragent legacy worker: printer is nil")
	}

	if cfg.UserID <= 0 {
		return nil, errors.New("printeragent legacy worker: user_id must be > 0")
	}

	cfg.WorkerID = strings.TrimSpace(cfg.WorkerID)
	if cfg.WorkerID == "" {
		return nil, errors.New("printeragent legacy worker: worker_id is required")
	}

	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.PollLimit <= 0 {
		cfg.PollLimit = 5
	}
	if cfg.ClaimTTL <= 0 {
		cfg.ClaimTTL = 2 * time.Minute
	}
	if cfg.PrintDelay < 0 {
		cfg.PrintDelay = 0
	}

	return &Worker{
		log:     log,
		cfg:     cfg,
		store:   store,
		printer: printer,
		now:     time.Now,
	}, nil
}

func (w *Worker) RunOnce(ctx context.Context) error {
	if w == nil {
		return errors.New("printeragent legacy worker: worker is nil")
	}

	if w.log != nil {
		w.log.Info("run once started",
			"user_id", w.cfg.UserID,
			"worker_id", w.cfg.WorkerID,
			"poll_limit", w.cfg.PollLimit,
			"claim_ttl", w.cfg.ClaimTTL,
			"print_delay", w.cfg.PrintDelay,
		)
	}

	return w.tick(ctx)
}

func (w *Worker) RunInterval(ctx context.Context) error {
	if w == nil {
		return errors.New("printeragent legacy worker: worker is nil")
	}

	if w.log != nil {
		w.log.Info("interval worker started",
			"user_id", w.cfg.UserID,
			"worker_id", w.cfg.WorkerID,
			"interval", w.cfg.Interval,
			"poll_limit", w.cfg.PollLimit,
			"claim_ttl", w.cfg.ClaimTTL,
			"print_delay", w.cfg.PrintDelay,
		)
	}

	if err := w.tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
		if w.log != nil {
			w.log.Warn("initial tick failed", "err", err)
		}
	}

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if w.log != nil {
				w.log.Info("interval worker stopped", "reason", ctx.Err())
			}
			return ctx.Err()

		case <-ticker.C:
			if err := w.tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
				if w.log != nil {
					w.log.Warn("tick failed", "err", err)
				}
			}
		}
	}
}

func (w *Worker) tick(ctx context.Context) error {
	now := w.now().UTC()

	jobs, err := w.store.ClaimReadyPrintJobs(ctx, storage.ClaimReadyPrintJobsInput{
		UserID: w.cfg.UserID,

		Now: now,

		Limit: w.cfg.PollLimit,

		WorkerID: w.cfg.WorkerID,
		ClaimTTL: w.cfg.ClaimTTL,
	})
	if err != nil {
		return fmt.Errorf("printeragent legacy worker: claim ready print jobs: %w", err)
	}

	if len(jobs) == 0 {
		if w.log != nil {
			w.log.Debug("no ready print jobs",
				"user_id", w.cfg.UserID,
				"now", now.Format(time.RFC3339),
			)
		}
		return nil
	}

	if w.log != nil {
		w.log.Info("print jobs claimed",
			"count", len(jobs),
			"user_id", w.cfg.UserID,
			"worker_id", w.cfg.WorkerID,
		)
	}

	for _, job := range jobs {
		if err := w.processJob(ctx, job); err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}

			if w.log != nil {
				w.log.Warn("process print job failed",
					"print_job_id", job.ID,
					"type", string(job.Type),
					"err", err,
				)
			}
		}

		if w.cfg.PrintDelay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(w.cfg.PrintDelay):
			}
		}
	}

	return nil
}

func (w *Worker) processJob(ctx context.Context, job storage.PrintJob) error {
	if job.ID <= 0 {
		return errors.New("printeragent legacy worker: print_job.id must be > 0")
	}

	if !job.IsClaimedBy(w.cfg.WorkerID) {
		return fmt.Errorf("printeragent legacy worker: print_job %d is not claimed by this worker", job.ID)
	}

	if !job.HasPayload() {
		return w.markFailed(ctx, job, "payload_text is empty")
	}

	result, err := w.printer.Print(ctx, job)
	if err != nil {
		markErr := w.markFailed(ctx, job, err.Error())
		if markErr != nil {
			return fmt.Errorf("print failed: %w; mark failed: %w", err, markErr)
		}

		return fmt.Errorf("print failed: %w", err)
	}

	printedJob, err := w.store.MarkPrintJobPrinted(ctx, storage.MarkPrintJobPrintedInput{
		PrintJobID: job.ID,

		WorkerID:  w.cfg.WorkerID,
		PrintedAt: w.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("mark print job printed: %w", err)
	}

	if w.log != nil {
		w.log.Info("print job completed",
			"print_job_id", printedJob.ID,
			"type", string(printedJob.Type),
			"destination", result.Destination,
			"bytes", result.Bytes,
			"printed_at", derefTime(printedJob.PrintedAt).Format(time.RFC3339),
		)
	}

	if printedJob.WakePlanID != nil && *printedJob.WakePlanID > 0 {
		if err := w.completeWakePlanIfPossible(ctx, *printedJob.WakePlanID); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) completeWakePlanIfPossible(ctx context.Context, wakePlanID int64) error {
	result, err := w.store.CompleteWakePlanIfPrinted(ctx, storage.CompleteWakePlanIfPrintedInput{
		WakePlanID: wakePlanID,
	})
	if errors.Is(err, storage.ErrNotFound) {
		if w.log != nil {
			w.log.Warn("wake plan not found during completion check",
				"wake_plan_id", wakePlanID,
			)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("complete wake plan if printed: %w", err)
	}

	if result.Completed {
		if w.log != nil {
			w.log.Info("wake plan completed",
				"wake_plan_id", result.WakePlanID,
				"status", result.Status,
			)
		}
		return nil
	}

	if w.log != nil {
		w.log.Debug("wake plan not completed yet",
			"wake_plan_id", result.WakePlanID,
			"status", result.Status,
		)
	}

	return nil
}

func (w *Worker) markFailed(ctx context.Context, job storage.PrintJob, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "printeragent legacy failed to print job"
	}

	failedJob, err := w.store.MarkPrintJobFailed(ctx, storage.MarkPrintJobFailedInput{
		PrintJobID: job.ID,

		WorkerID: w.cfg.WorkerID,

		ErrorMessage: message,
		FailedAt:     w.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("mark print job failed: %w", err)
	}

	if w.log != nil {
		w.log.Warn("print job marked failed",
			"print_job_id", failedJob.ID,
			"type", string(failedJob.Type),
			"error_message", message,
		)
	}

	return nil
}

func derefTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}

	return *value
}
