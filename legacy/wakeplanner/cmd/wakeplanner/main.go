package main

import (
	"fmt"
	"os"

	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/app"
	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/banner"
	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/config"
	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/lifecycle"
	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/logger"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "wakeplanner-legacy failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, configPath, err := config.Load(os.Getenv("WAKEPLANNER_CONFIG"))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log, err := logger.New(logger.Options{
		Level:       cfg.Logger.Level,
		FileEnabled: cfg.Logger.FileEnabled,
		FilePath:    cfg.Logger.FilePath,
	})
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() {
		if err := log.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "wakeplanner-legacy logger close failed: %v\n", err)
		}
	}()

	banner.Print(os.Stdout, banner.Info{
		Version: version,

		UserID: cfg.WakePlanner.UserID,

		GatewayURL: cfg.Storage.HTTP.BaseURL,

		Lookahead:   cfg.WakePlanner.Lookahead.Duration,
		PreWakeLead: cfg.WakePlanner.PreWakeLead.Duration,

		SleepAfterPlanning: cfg.WakePlanner.SleepAfterPlanning,
		DryRun:             cfg.WakePlanner.DryRun,

		LogFile: logFilePath(cfg),
	})

	log.Info("legacy wakeplanner booting",
		"version", version,
		"commit", commit,
		"build_date", buildDate,
		"app", cfg.AppName,
		"env", cfg.Env,
		"config_path", configPath,
		"printergateway_base_url", cfg.Storage.HTTP.BaseURL,
		"user_id", cfg.WakePlanner.UserID,
		"lookahead", cfg.WakePlanner.Lookahead.Duration,
		"pre_wake_lead", cfg.WakePlanner.PreWakeLead.Duration,
		"sleep_after_planning", cfg.WakePlanner.SleepAfterPlanning,
		"dry_run", cfg.WakePlanner.DryRun,
	)

	application, err := app.New(cfg, log)
	if err != nil {
		log.Error("app init failed", "err", err)
		return fmt.Errorf("app init: %w", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Error("app close failed", "err", err)
		}
	}()

	if err := lifecycle.RunWithGracefulShutdown(application, log, lifecycle.Options{
		ShutdownTimeout: cfg.Runtime.ShutdownTimeout.Duration,

		DryRun:             cfg.WakePlanner.DryRun,
		SleepAfterPlanning: cfg.WakePlanner.SleepAfterPlanning,
		GatewayURL:         cfg.Storage.HTTP.BaseURL,
		LogFile:            logFilePath(cfg),

		Out: os.Stdout,
	}); err != nil {
		log.Error("app run failed", "err", err)
		return fmt.Errorf("app run: %w", err)
	}

	log.Info("legacy wakeplanner stopped")

	return nil
}

func logFilePath(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}

	if !cfg.Logger.FileEnabled {
		return ""
	}

	return cfg.Logger.FilePath
}
