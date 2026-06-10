package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/config"
	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/logger"
	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/planner"
	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/pmset"
	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/storage"
)

type App struct {
	cfg *config.Config
	log *logger.Logger

	store storage.Store
	plan  *planner.Planner
}

func New(cfg *config.Config, log *logger.Logger) (*App, error) {
	if cfg == nil {
		return nil, errors.New("wakeplanner legacy app: config is nil")
	}
	if log == nil {
		return nil, errors.New("wakeplanner legacy app: logger is nil")
	}

	agentName := defaultAgentName()

	st, err := openStore(cfg, log, agentName)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	power, err := pmset.New(pmset.Config{
		Path:   cfg.PMSet.Path,
		DryRun: cfg.WakePlanner.DryRun,
	})
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create pmset client: %w", err)
	}

	plannerInstance, err := planner.New(log, planner.Config{
		UserID: cfg.WakePlanner.UserID,

		Lookahead:   cfg.WakePlanner.Lookahead.Duration,
		PreWakeLead: cfg.WakePlanner.PreWakeLead.Duration,

		SleepAfterPlanning: cfg.WakePlanner.SleepAfterPlanning,
	}, st, power)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create planner: %w", err)
	}

	log.Info("wakeplanner legacy initialized",
		"user_id", cfg.WakePlanner.UserID,
		"storage", "http",
		"printergateway_base_url", cfg.Storage.HTTP.BaseURL,
		"token_source", tokenSourceLabel(cfg),
		"agent_name", agentName,
		"lookahead", cfg.WakePlanner.Lookahead.Duration,
		"pre_wake_lead", cfg.WakePlanner.PreWakeLead.Duration,
		"sleep_after_planning", cfg.WakePlanner.SleepAfterPlanning,
		"dry_run", cfg.WakePlanner.DryRun,
		"pmset_path", cfg.PMSet.Path,
	)

	return &App{
		cfg:   cfg,
		log:   log,
		store: st,
		plan:  plannerInstance,
	}, nil
}

func openStore(cfg *config.Config, log *logger.Logger, agentName string) (storage.Store, error) {
	if cfg == nil {
		return nil, errors.New("wakeplanner legacy app: config is nil")
	}

	token, err := loadPrinterGatewayToken(cfg)
	if err != nil {
		return nil, fmt.Errorf("load printergateway token: %w", err)
	}

	st, err := storage.NewHTTP(storage.HTTPConfig{
		BaseURL: cfg.Storage.HTTP.BaseURL,
		Token:   token,
		Timeout: cfg.Storage.HTTP.Timeout.Duration,

		AgentName: agentName,
	})
	if err != nil {
		return nil, fmt.Errorf("create printergateway storage: %w", err)
	}

	if log != nil {
		log.Info("printergateway storage initialized",
			"base_url", cfg.Storage.HTTP.BaseURL,
			"timeout", cfg.Storage.HTTP.Timeout.Duration,
			"token_source", tokenSourceLabel(cfg),
			"agent_name", agentName,
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
		return errors.New("wakeplanner legacy app: app is nil")
	}
	if a.plan == nil {
		return errors.New("wakeplanner legacy app: planner is nil")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	a.log.Info("run started",
		"user_id", a.cfg.WakePlanner.UserID,
		"lookahead", a.cfg.WakePlanner.Lookahead.Duration,
		"pre_wake_lead", a.cfg.WakePlanner.PreWakeLead.Duration,
		"sleep_after_planning", a.cfg.WakePlanner.SleepAfterPlanning,
		"dry_run", a.cfg.WakePlanner.DryRun,
		"printergateway_base_url", a.cfg.Storage.HTTP.BaseURL,
		"pmset_path", a.cfg.PMSet.Path,
	)

	result, err := a.plan.RunOnce(ctx)
	if err != nil {
		return fmt.Errorf("run planner once: %w", err)
	}

	if !result.PlanFound {
		a.log.Info("run completed: no wake plan found")
		return nil
	}

	a.log.Info("run completed: wake plan scheduled",
		"wake_plan_id", result.WakePlanID,
		"wake_at_utc", result.WakeAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"wake_at_local", result.WakeAt.Local().Format("2006-01-02T15:04:05Z07:00"),
		"scheduled_wake_at_utc", result.ScheduledWakeAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"scheduled_wake_at_local", result.ScheduledWakeAt.Local().Format("2006-01-02T15:04:05Z07:00"),
	)

	return nil
}

func (a *App) Close() error {
	if a == nil || a.store == nil {
		return nil
	}

	return a.store.Close()
}

func defaultAgentName() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "unknown-host"
	}

	return fmt.Sprintf("wakeplanner-legacy-%s-%d", hostname, os.Getpid())
}
