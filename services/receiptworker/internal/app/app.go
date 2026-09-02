package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	platformpg "github.com/faringet/whoop-morning-printer/internal/platform/postgres"
	"github.com/faringet/whoop-morning-printer/services/receiptworker/config"
	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/art"
	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/fieldnote"
	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/storage"
	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/worker"
)

type App struct {
	cfg *config.Config
	log *slog.Logger

	store              storage.Store
	fieldNoteGenerator fieldnote.Generator
	worker             *worker.Worker
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	if cfg == nil {
		return nil, errors.New("receiptworker app: config is nil")
	}
	if log == nil {
		return nil, errors.New("receiptworker app: logger is nil")
	}

	appLog := log.With(
		slog.String("layer", "app"),
		slog.String("module", "receiptworker.app"),
	)

	st, err := openStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	templates, err := art.LoadEmbeddedTemplates()
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("load art templates: %w", err)
	}

	artSelector := art.NewSelector(templates)
	fieldNoteGenerator := buildFieldNoteGenerator(cfg)

	w, err := worker.New(log, worker.Config{
		UserID: cfg.ReceiptWorker.UserID,

		Timezone: cfg.ReceiptWorker.Timezone,

		Interval:  cfg.ReceiptWorker.Interval,
		PollLimit: cfg.ReceiptWorker.PollLimit,

		ProcessWakeReceipt: cfg.ReceiptWorker.ShouldProcessWakeReceipt(),
		ProcessFinalReport: cfg.ReceiptWorker.ShouldProcessFinalReport(),

		EnsureFinalReportJobs: cfg.ReceiptWorker.ShouldEnsureFinalReportJobs(),

		FinalReportRequireAdvice: cfg.ReceiptWorker.ShouldRequireAdviceForFinalReport(),
		FallbackAfterDeadline:    cfg.ReceiptWorker.ShouldFallbackAfterDeadline(),

		ReceiptWidth:         cfg.Receipt.Width,
		ReceiptLineSeparator: cfg.Receipt.LineSeparator,

		ArtEnabled:  cfg.Receipt.IsArtEnabled(),
		ArtMode:     cfg.Receipt.ArtMode,
		ArtMaxLines: cfg.Receipt.MaxArtLines,
	}, st, artSelector, fieldNoteGenerator)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create worker: %w", err)
	}

	appLog.Info("art templates loaded",
		slog.Int("count", len(templates)),
	)

	return &App{
		cfg:                cfg,
		log:                appLog,
		store:              st,
		fieldNoteGenerator: fieldNoteGenerator,
		worker:             w,
	}, nil
}

func buildFieldNoteGenerator(cfg *config.Config) fieldnote.Generator {
	if !cfg.FieldNote.IsEnabled() {
		return nil
	}

	generators := make([]fieldnote.Generator, 0, 3)

	if cfg.FieldNote.Ollama.IsEnabled() {
		ollamaGenerator := fieldnote.NewOllamaGenerator(cfg.FieldNote.Ollama.BaseURL, cfg.FieldNote.Ollama.Model, cfg.FieldNote.Ollama.KeepAlive, cfg.FieldNote.Ollama.Timeout)
		generators = append(generators, ollamaGenerator)
	}

	grammarGenerator := fieldnote.NewGrammarGenerator()
	generators = append(generators, grammarGenerator)

	emergencyGenerator := fieldnote.NewEmergencyGenerator()
	generators = append(generators, emergencyGenerator)

	return fieldnote.NewResilientGenerator(generators...)
}

func openStore(cfg *config.Config) (storage.Store, error) {
	if cfg == nil {
		return nil, errors.New("receiptworker app: config is nil")
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
		return errors.New("receiptworker app: app is nil")
	}
	if a.worker == nil {
		return errors.New("receiptworker app: worker is nil")
	}

	mode := strings.ToLower(strings.TrimSpace(a.cfg.ReceiptWorker.Mode))
	if mode == "" {
		mode = "once"
	}

	a.log.Info("run started",
		slog.String("mode", mode),
		slog.Int64("user_id", a.cfg.ReceiptWorker.UserID),
		slog.String("timezone", a.cfg.ReceiptWorker.Timezone),
		slog.Duration("interval", a.cfg.ReceiptWorker.Interval),
		slog.Int("poll_limit", a.cfg.ReceiptWorker.PollLimit),
		slog.Bool("process_wake_receipt", a.cfg.ReceiptWorker.ShouldProcessWakeReceipt()),
		slog.Bool("process_final_report", a.cfg.ReceiptWorker.ShouldProcessFinalReport()),
		slog.Bool("ensure_final_report_jobs", a.cfg.ReceiptWorker.ShouldEnsureFinalReportJobs()),
		slog.Bool("final_report_require_advice", a.cfg.ReceiptWorker.ShouldRequireAdviceForFinalReport()),
		slog.Bool("fallback_after_deadline", a.cfg.ReceiptWorker.ShouldFallbackAfterDeadline()),
		slog.Bool("field_note_enabled", a.cfg.FieldNote.IsEnabled()),
		slog.Bool("field_note_ollama_enabled", a.cfg.FieldNote.Ollama.IsEnabled()),
		slog.Int("receipt_width", a.cfg.Receipt.Width),
		slog.Bool("art_enabled", a.cfg.Receipt.IsArtEnabled()),
		slog.String("art_mode", a.cfg.Receipt.ArtMode),
		slog.Int("art_max_lines", a.cfg.Receipt.MaxArtLines),
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
		return fmt.Errorf("receiptworker app: unsupported receiptworker.mode %q", a.cfg.ReceiptWorker.Mode)
	}
}

func (a *App) Close() error {
	if a == nil || a.store == nil {
		return nil
	}

	return a.store.Close()
}
