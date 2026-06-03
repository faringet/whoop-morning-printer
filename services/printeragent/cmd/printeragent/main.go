package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/faringet/whoop-morning-printer/pkg/logger"
	"github.com/faringet/whoop-morning-printer/services/printeragent/config"
	"github.com/faringet/whoop-morning-printer/services/printeragent/internal/app"
	"github.com/faringet/whoop-morning-printer/services/printeragent/internal/banner"
)

func main() {
	cfg := config.New()

	log := logger.NewLogger(logger.Options{
		AppName:     cfg.Base.AppName,
		Env:         cfg.Base.Env,
		Level:       cfg.Logger.Level,
		JSON:        cfg.Logger.JSON,
		FileEnabled: cfg.Logger.FileEnabled,
		FilePath:    cfg.Logger.FilePath,
	})

	banner.Print(os.Stdout, banner.Info{
		Mode:        cfg.PrinterAgent.Mode,
		OutputMode:  cfg.Output.Mode,
		PrinterName: cfg.Output.PrinterName,
		CPI:         cfg.Output.CPI,
		LPI:         cfg.Output.LPI,
		DatabaseDSN: cfg.Storage.Postgres.DSN,
		LogFile:     logFilePath(cfg),
	})

	application, err := app.New(cfg, log)
	if err != nil {
		log.Error("app init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Error("app close failed", slog.Any("err", err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := application.Run(ctx); err != nil && !isShutdownErr(err) {
		log.Error("app run failed", slog.Any("err", err))
		os.Exit(1)
	}
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
