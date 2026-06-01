package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	platformpg "github.com/faringet/whoop-morning-printer/internal/platform/postgres"
	"github.com/faringet/whoop-morning-printer/services/printeragent/config"
	"github.com/faringet/whoop-morning-printer/services/printeragent/internal/output"
	"github.com/faringet/whoop-morning-printer/services/printeragent/internal/storage"
	"github.com/faringet/whoop-morning-printer/services/printeragent/internal/worker"
)

type App struct {
	cfg *config.Config
	log *slog.Logger

	store  storage.Store
	worker *worker.Worker
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	if cfg == nil {
		return nil, errors.New("printeragent app: config is nil")
	}
	if log == nil {
		return nil, errors.New("printeragent app: logger is nil")
	}

	appLog := log.With(
		slog.String("layer", "app"),
		slog.String("module", "printeragent.app"),
	)

	st, err := openStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	workerID := strings.TrimSpace(cfg.PrinterAgent.WorkerID)
	if workerID == "" {
		workerID = defaultWorkerID()
	}

	printer, err := output.New(log, output.Config{
		Mode: cfg.Output.Mode,

		Dir:        cfg.Output.Dir,
		CreateDirs: cfg.Output.ShouldCreateDirs(),

		TrailingBlankLines: cfg.Output.TrailingBlankLines,
	})
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create output printer: %w", err)
	}

	w, err := worker.New(log, worker.Config{
		UserID: cfg.PrinterAgent.UserID,

		Interval:  cfg.PrinterAgent.Interval,
		PollLimit: cfg.PrinterAgent.PollLimit,

		WorkerID: workerID,
		ClaimTTL: cfg.PrinterAgent.ClaimTTL,

		PrintDelay: cfg.PrinterAgent.PrintDelay,
	}, st, printer)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create worker: %w", err)
	}

	appLog.Info("printeragent initialized",
		slog.String("worker_id", workerID),
		slog.String("output_mode", cfg.Output.Mode),
		slog.String("output_dir", cfg.Output.Dir),
	)

	return &App{
		cfg:    cfg,
		log:    appLog,
		store:  st,
		worker: w,
	}, nil
}

func openStore(cfg *config.Config) (storage.Store, error) {
	if cfg == nil {
		return nil, errors.New("printeragent app: config is nil")
	}

	db, err := platformpg.Open(platformpg.Config{
		DSN:             cfg.Storage.Postgres.DSN,
		MaxOpenConns:    cfg.Storage.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Storage.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Storage.Postgres.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Storage.Postgres.ConnMaxIdleTime,
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres db: %w", err)
	}

	st, err := storage.NewPostgres(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create postgres storage: %w", err)
	}

	return st, nil
}

func (a *App) Run(ctx context.Context) error {
	if a == nil {
		return errors.New("printeragent app: app is nil")
	}
	if a.worker == nil {
		return errors.New("printeragent app: worker is nil")
	}

	mode := strings.ToLower(strings.TrimSpace(a.cfg.PrinterAgent.Mode))
	if mode == "" {
		mode = "once"
	}

	a.log.Info("run started",
		slog.String("mode", mode),
		slog.Int64("user_id", a.cfg.PrinterAgent.UserID),
		slog.Duration("interval", a.cfg.PrinterAgent.Interval),
		slog.Int("poll_limit", a.cfg.PrinterAgent.PollLimit),
		slog.Duration("claim_ttl", a.cfg.PrinterAgent.ClaimTTL),
		slog.Duration("print_delay", a.cfg.PrinterAgent.PrintDelay),
		slog.String("output_mode", a.cfg.Output.Mode),
		slog.String("output_dir", a.cfg.Output.Dir),
		slog.Bool("output_create_dirs", a.cfg.Output.ShouldCreateDirs()),
		slog.Int("trailing_blank_lines", a.cfg.Output.TrailingBlankLines),
	)

	a.log.Info("postgres storage configured",
		slog.Int("max_open_conns", a.cfg.Storage.Postgres.MaxOpenConns),
		slog.Int("max_idle_conns", a.cfg.Storage.Postgres.MaxIdleConns),
	)

	switch mode {
	case "once":
		return a.worker.RunOnce(ctx)

	case "interval":
		return a.worker.RunInterval(ctx)

	default:
		return fmt.Errorf("printeragent app: unsupported printeragent.mode %q", a.cfg.PrinterAgent.Mode)
	}
}

func (a *App) Close() error {
	if a == nil || a.store == nil {
		return nil
	}

	return a.store.Close()
}

func defaultWorkerID() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "unknown-host"
	}

	return fmt.Sprintf("printeragent-%s-%d", hostname, os.Getpid())
}
