package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/config"
	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/logger"
	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/output"
	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/storage"
	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/worker"
)

type App struct {
	cfg *config.Config
	log *logger.Logger

	store  storage.Store
	worker *worker.Worker
}

func New(cfg *config.Config, log *logger.Logger) (*App, error) {
	if cfg == nil {
		return nil, errors.New("printeragent legacy app: config is nil")
	}
	if log == nil {
		return nil, errors.New("printeragent legacy app: logger is nil")
	}

	st, err := openStore(cfg, log)
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

		PrinterName: cfg.Output.PrinterName,
		CPI:         cfg.Output.CPI,
		LPI:         cfg.Output.LPI,

		SpoolDir:       cfg.Output.SpoolDir,
		KeepSpoolFiles: cfg.Output.ShouldKeepSpoolFiles(),
	})
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create output printer: %w", err)
	}

	w, err := worker.New(log, worker.Config{
		UserID: cfg.PrinterAgent.UserID,

		Interval:  cfg.PrinterAgent.Interval.Duration,
		PollLimit: cfg.PrinterAgent.PollLimit,

		WorkerID: workerID,
		ClaimTTL: cfg.PrinterAgent.ClaimTTL.Duration,

		PrintDelay: cfg.PrinterAgent.PrintDelay.Duration,
	}, st, printer)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create worker: %w", err)
	}

	log.Info("printeragent legacy initialized",
		"worker_id", workerID,
		"output_mode", cfg.Output.Mode,
		"output_dir", cfg.Output.Dir,
		"output_printer_name", cfg.Output.PrinterName,
		"output_cpi", cfg.Output.CPI,
		"output_lpi", cfg.Output.LPI,
		"output_spool_dir", cfg.Output.SpoolDir,
		"output_keep_spool_files", cfg.Output.ShouldKeepSpoolFiles(),
	)

	return &App{
		cfg:    cfg,
		log:    log,
		store:  st,
		worker: w,
	}, nil
}

func openStore(cfg *config.Config, log *logger.Logger) (storage.Store, error) {
	if cfg == nil {
		return nil, errors.New("printeragent legacy app: config is nil")
	}

	db, err := sql.Open("postgres", cfg.Storage.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres db: %w", err)
	}

	db.SetMaxOpenConns(cfg.Storage.Postgres.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Storage.Postgres.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Storage.Postgres.ConnMaxLifetime.Duration)
	db.SetConnMaxIdleTime(cfg.Storage.Postgres.ConnMaxIdleTime.Duration)

	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres db: %w", err)
	}

	if log != nil {
		var now time.Time
		var currentUser string
		var currentDatabase string

		err := db.QueryRowContext(pingCtx, `
			SELECT
				NOW(),
				CURRENT_USER,
				CURRENT_DATABASE()
		`).Scan(&now, &currentUser, &currentDatabase)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("postgres health query: %w", err)
		}

		log.Info("postgres connected",
			"current_user", currentUser,
			"current_database", currentDatabase,
			"db_time_utc", now.UTC().Format(time.RFC3339),
			"max_open_conns", cfg.Storage.Postgres.MaxOpenConns,
			"max_idle_conns", cfg.Storage.Postgres.MaxIdleConns,
		)
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
		return errors.New("printeragent legacy app: app is nil")
	}
	if a.worker == nil {
		return errors.New("printeragent legacy app: worker is nil")
	}

	mode := strings.ToLower(strings.TrimSpace(a.cfg.PrinterAgent.Mode))
	if mode == "" {
		mode = "once"
	}

	a.log.Info("run started",
		"mode", mode,
		"user_id", a.cfg.PrinterAgent.UserID,
		"interval", a.cfg.PrinterAgent.Interval.Duration,
		"poll_limit", a.cfg.PrinterAgent.PollLimit,
		"claim_ttl", a.cfg.PrinterAgent.ClaimTTL.Duration,
		"print_delay", a.cfg.PrinterAgent.PrintDelay.Duration,
		"output_mode", a.cfg.Output.Mode,
		"output_dir", a.cfg.Output.Dir,
		"output_create_dirs", a.cfg.Output.ShouldCreateDirs(),
		"trailing_blank_lines", a.cfg.Output.TrailingBlankLines,
		"output_printer_name", a.cfg.Output.PrinterName,
		"output_cpi", a.cfg.Output.CPI,
		"output_lpi", a.cfg.Output.LPI,
		"output_spool_dir", a.cfg.Output.SpoolDir,
		"output_keep_spool_files", a.cfg.Output.ShouldKeepSpoolFiles(),
	)

	switch mode {
	case "once":
		return a.worker.RunOnce(ctx)

	case "interval":
		return a.worker.RunInterval(ctx)

	default:
		return fmt.Errorf("printeragent legacy app: unsupported printeragent.mode %q", a.cfg.PrinterAgent.Mode)
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

	return fmt.Sprintf("printeragent-legacy-%s-%d", hostname, os.Getpid())
}
