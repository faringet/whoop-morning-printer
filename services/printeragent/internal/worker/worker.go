package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/printeragent/internal/output"
	"github.com/faringet/whoop-morning-printer/services/printeragent/internal/storage"
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
	log *slog.Logger
	cfg Config

	store   storage.Store
	printer output.Printer

	now func() time.Time
}

func New(
	log *slog.Logger,
	cfg Config,
	store storage.Store,
	printer output.Printer,
) (*Worker, error) {
	if log == nil {
		log = slog.Default()
	}
	if store == nil {
		return nil, errors.New("printeragent worker: store is nil")
	}
	if printer == nil {
		return nil, errors.New("printeragent worker: printer is nil")
	}

	if cfg.UserID <= 0 {
		return nil, errors.New("printeragent worker: user_id must be > 0")
	}

	cfg.WorkerID = strings.TrimSpace(cfg.WorkerID)
	if cfg.WorkerID == "" {
		return nil, errors.New("printeragent worker: worker_id is required")
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
		log: log.With(
			slog.String("layer", "worker"),
			slog.String("module", "printeragent.worker"),
		),
		cfg:     cfg,
		store:   store,
		printer: printer,
		now:     time.Now,
	}, nil
}

func (w *Worker) RunOnce(ctx context.Context) error {
	if w == nil {
		return errors.New("printeragent worker: worker is nil")
	}

	w.log.Info("run once started",
		slog.Int64("user_id", w.cfg.UserID),
		slog.String("worker_id", w.cfg.WorkerID),
		slog.Int("poll_limit", w.cfg.PollLimit),
		slog.Duration("claim_ttl", w.cfg.ClaimTTL),
		slog.Duration("print_delay", w.cfg.PrintDelay),
	)

	return w.tick(ctx)
}

func (w *Worker) RunInterval(ctx context.Context) error {
	if w == nil {
		return errors.New("printeragent worker: worker is nil")
	}

	w.log.Info("interval worker started",
		slog.Int64("user_id", w.cfg.UserID),
		slog.String("worker_id", w.cfg.WorkerID),
		slog.Duration("interval", w.cfg.Interval),
		slog.Int("poll_limit", w.cfg.PollLimit),
		slog.Duration("claim_ttl", w.cfg.ClaimTTL),
		slog.Duration("print_delay", w.cfg.PrintDelay),
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

	jobs, err := w.store.ClaimReadyPrintJobs(ctx, storage.ClaimReadyPrintJobsInput{
		UserID: w.cfg.UserID,

		Now: now,

		Limit: w.cfg.PollLimit,

		WorkerID: w.cfg.WorkerID,
		ClaimTTL: w.cfg.ClaimTTL,
	})
	if err != nil {
		return fmt.Errorf("printeragent worker: claim ready print jobs: %w", err)
	}

	if len(jobs) == 0 {
		w.log.Debug("no ready print jobs",
			slog.Int64("user_id", w.cfg.UserID),
			slog.Time("now", now),
		)
		return nil
	}

	w.log.Info("print jobs claimed",
		slog.Int("count", len(jobs)),
		slog.Int64("user_id", w.cfg.UserID),
		slog.String("worker_id", w.cfg.WorkerID),
	)

	for _, job := range jobs {
		if err := w.processJob(ctx, job); err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}

			w.log.Warn("process print job failed",
				slog.Int64("print_job_id", job.ID),
				slog.String("type", string(job.Type)),
				slog.Any("err", err),
			)
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
		return errors.New("printeragent worker: print_job.id must be > 0")
	}

	if !job.IsClaimedBy(w.cfg.WorkerID) {
		return fmt.Errorf("printeragent worker: print_job %d is not claimed by this worker", job.ID)
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

	w.log.Info("print job completed",
		slog.Int64("print_job_id", printedJob.ID),
		slog.String("type", string(printedJob.Type)),
		slog.String("destination", result.Destination),
		slog.Int("bytes", result.Bytes),
		slog.Time("printed_at", derefTime(printedJob.PrintedAt)),
	)

	return nil
}

func (w *Worker) markFailed(ctx context.Context, job storage.PrintJob, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "printeragent failed to print job"
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

	w.log.Warn("print job marked failed",
		slog.Int64("print_job_id", failedJob.ID),
		slog.String("type", string(failedJob.Type)),
		slog.String("error_message", message),
	)

	return nil
}

func derefTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}

	return *value
}
