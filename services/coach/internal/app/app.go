package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	platformpg "github.com/faringet/whoop-morning-printer/internal/platform/postgres"
	"github.com/faringet/whoop-morning-printer/services/coach/config"
	"github.com/faringet/whoop-morning-printer/services/coach/internal/advisor"
	"github.com/faringet/whoop-morning-printer/services/coach/internal/ollama"
	"github.com/faringet/whoop-morning-printer/services/coach/internal/storage"
	"github.com/faringet/whoop-morning-printer/services/coach/internal/worker"
)

type App struct {
	cfg *config.Config
	log *slog.Logger

	store  storage.Store
	worker *worker.Worker
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	if cfg == nil {
		return nil, errors.New("coach app: config is nil")
	}
	if log == nil {
		return nil, errors.New("coach app: logger is nil")
	}

	appLog := log.With(
		slog.String("layer", "app"),
		slog.String("module", "coach.app"),
	)

	st, err := openStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	ollamaClient, err := ollama.NewClient(ollama.Config{
		BaseURL:   cfg.Ollama.BaseURL,
		Timeout:   cfg.Ollama.Timeout,
		KeepAlive: cfg.Ollama.KeepAlive,
		Think:     cfg.Ollama.Think,
	})
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create ollama client: %w", err)
	}

	adv, err := advisor.New(appLog, advisor.Config{
		Model: cfg.Ollama.Model,

		PromptVersion: cfg.Coach.PromptVersion,
		PromptPath:    cfg.Coach.PromptPath,

		MaxRetries:   cfg.Coach.MaxRetries,
		RetryBackoff: cfg.Coach.RetryBackoff,

		MaxAdviceRunes: cfg.Coach.MaxAdviceRunes,
		MaxMottoRunes:  cfg.Coach.MaxMottoRunes,
	}, ollamaClient)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create advisor: %w", err)
	}

	w, err := worker.New(appLog, worker.Config{
		UserID: cfg.Coach.UserID,

		Timezone: cfg.Coach.Timezone,

		Model:         cfg.Ollama.Model,
		PromptVersion: cfg.Coach.PromptVersion,

		Interval:     cfg.Coach.Interval,
		PollInterval: cfg.Coach.PollInterval,

		SnapshotLookbackDays: cfg.Coach.SnapshotLookbackDays,

		RequireReadySnapshot:      cfg.Coach.RequireReadySnapshot,
		AllowPartialAfterDeadline: cfg.Coach.AllowPartialAfterDeadline,

		WarmupOnStart:       cfg.Coach.WarmupOnStart,
		WarmupBeforeWake:    cfg.Coach.WarmupBeforeWake,
		WarmupTimeout:       cfg.Coach.WarmupTimeout,
		MinWarmupInterval:   cfg.Coach.MinWarmupInterval,
		ActiveWakePlanLimit: 10,
	}, st, adv, ollamaClient)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create worker: %w", err)
	}

	return &App{
		cfg:    cfg,
		log:    appLog,
		store:  st,
		worker: w,
	}, nil
}

func openStore(cfg *config.Config) (storage.Store, error) {
	if cfg == nil {
		return nil, errors.New("coach app: config is nil")
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
		return errors.New("coach app: app is nil")
	}
	if a.worker == nil {
		return errors.New("coach app: worker is nil")
	}

	mode := strings.ToLower(strings.TrimSpace(a.cfg.Coach.Mode))
	if mode == "" {
		mode = "wake_watch"
	}

	a.log.Info("run started",
		slog.String("mode", mode),
		slog.Int64("user_id", a.cfg.Coach.UserID),
		slog.String("timezone", a.cfg.Coach.Timezone),
		slog.String("ollama_url", a.cfg.Ollama.BaseURL),
		slog.String("model", a.cfg.Ollama.Model),
		slog.String("keep_alive", a.cfg.Ollama.KeepAlive),
		slog.Bool("think", a.cfg.Ollama.Think),
		slog.String("prompt_version", a.cfg.Coach.PromptVersion),
		slog.Bool("warmup_on_start", a.cfg.Coach.WarmupOnStart),
		slog.Bool("require_ready_snapshot", a.cfg.Coach.RequireReadySnapshot),
		slog.Bool("allow_partial_after_deadline", a.cfg.Coach.AllowPartialAfterDeadline),
	)

	a.log.Info("postgres storage configured",
		slog.Int("max_open_conns", a.cfg.Storage.Postgres.MaxOpenConns),
		slog.Int("max_idle_conns", a.cfg.Storage.Postgres.MaxIdleConns),
	)

	switch mode {
	case "once":
		return a.worker.RunOnce(ctx)

	case "interval":
		a.log.Info("interval mode configured",
			slog.Duration("interval", a.cfg.Coach.Interval),
			slog.Int("snapshot_lookback_days", a.cfg.Coach.SnapshotLookbackDays),
		)

		return a.worker.RunInterval(ctx)

	case "wake_watch":
		a.log.Info("wake_watch mode configured",
			slog.Duration("poll_interval", a.cfg.Coach.PollInterval),
			slog.Duration("warmup_before_wake", a.cfg.Coach.WarmupBeforeWake),
			slog.Duration("warmup_timeout", a.cfg.Coach.WarmupTimeout),
			slog.Duration("min_warmup_interval", a.cfg.Coach.MinWarmupInterval),
			slog.Int("snapshot_lookback_days", a.cfg.Coach.SnapshotLookbackDays),
		)

		return a.worker.RunWakeWatch(ctx)

	default:
		return fmt.Errorf("coach app: unsupported coach.mode %q", a.cfg.Coach.Mode)
	}
}

func (a *App) Close() error {
	if a == nil || a.store == nil {
		return nil
	}

	return a.store.Close()
}
