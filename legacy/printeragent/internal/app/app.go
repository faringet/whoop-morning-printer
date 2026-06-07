package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

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

	workerID := strings.TrimSpace(cfg.PrinterAgent.WorkerID)
	if workerID == "" {
		workerID = defaultWorkerID()
	}

	st, err := openStore(cfg, log, workerID)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
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
		"storage", "http",
		"printergateway_base_url", cfg.Storage.HTTP.BaseURL,
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

func openStore(cfg *config.Config, log *logger.Logger, workerID string) (storage.Store, error) {
	if cfg == nil {
		return nil, errors.New("printeragent legacy app: config is nil")
	}

	token, err := loadPrinterGatewayToken(cfg)
	if err != nil {
		return nil, fmt.Errorf("load printergateway token: %w", err)
	}

	st, err := storage.NewHTTP(storage.HTTPConfig{
		BaseURL: cfg.Storage.HTTP.BaseURL,
		Token:   token,
		Timeout: cfg.Storage.HTTP.Timeout.Duration,

		UserID:   cfg.PrinterAgent.UserID,
		WorkerID: workerID,
	})
	if err != nil {
		return nil, fmt.Errorf("create printergateway storage: %w", err)
	}

	if log != nil {
		log.Info("printergateway storage initialized",
			"base_url", cfg.Storage.HTTP.BaseURL,
			"timeout", cfg.Storage.HTTP.Timeout.Duration,
			"user_id", cfg.PrinterAgent.UserID,
			"worker_id", workerID,
			"token_source", tokenSourceLabel(cfg),
		)
	}

	return st, nil
}

func loadPrinterGatewayToken(cfg *config.Config) (string, error) {
	if cfg == nil {
		return "", errors.New("config is nil")
	}

	token := strings.TrimSpace(cfg.Storage.HTTP.Token)
	if token != "" {
		return token, nil
	}

	tokenFile := strings.TrimSpace(cfg.Storage.HTTP.TokenFile)
	if tokenFile == "" {
		return "", errors.New("storage.http.token or storage.http.token_file is required")
	}

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("read token file %q: %w", tokenFile, err)
	}

	token = strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("token file %q is empty", tokenFile)
	}

	return token, nil
}

func tokenSourceLabel(cfg *config.Config) string {
	if cfg == nil {
		return "-"
	}

	if strings.TrimSpace(cfg.Storage.HTTP.Token) != "" {
		return "config"
	}

	if strings.TrimSpace(cfg.Storage.HTTP.TokenFile) != "" {
		return "token_file"
	}

	return "-"
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
		"storage", "http",
		"printergateway_base_url", a.cfg.Storage.HTTP.BaseURL,
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
