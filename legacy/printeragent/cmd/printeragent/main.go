package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/app"
	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/banner"
	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/config"
	"github.com/faringet/whoop-morning-printer/legacy/printeragent/internal/logger"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "printeragent-legacy failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, configPath, err := config.Load(os.Getenv("PRINTERAGENT_CONFIG"))
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
			fmt.Fprintf(os.Stderr, "printeragent-legacy logger close failed: %v\n", err)
		}
	}()

	banner.Print(os.Stdout, banner.Info{
		Version: version,

		Mode:        cfg.PrinterAgent.Mode,
		OutputMode:  cfg.Output.Mode,
		PrinterName: cfg.Output.PrinterName,
		CPI:         cfg.Output.CPI,
		LPI:         cfg.Output.LPI,

		DatabaseDSN: cfg.Storage.Postgres.DSN,
		LogFile:     logFilePath(cfg),
	})

	log.Info("legacy printeragent booting",
		"version", version,
		"commit", commit,
		"build_date", buildDate,
		"app", cfg.AppName,
		"env", cfg.Env,
		"config_path", configPath,
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := application.Run(ctx); err != nil && !isShutdownErr(err) {
		log.Error("app run failed", "err", err)
		return fmt.Errorf("app run: %w", err)
	}

	log.Info("legacy printeragent stopped")

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

func isShutdownErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
