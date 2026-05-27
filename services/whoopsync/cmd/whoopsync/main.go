package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faringet/whoop-morning-printer/pkg/logger"
	"github.com/faringet/whoop-morning-printer/services/whoopsync/config"
	"github.com/faringet/whoop-morning-printer/services/whoopsync/internal/app"
)

func main() {
	cfg := config.New()

	log := logger.NewLogger(logger.Options{
		AppName: cfg.Base.AppName,
		Env:     cfg.Base.Env,
		Level:   cfg.Logger.Level,
		JSON:    cfg.Logger.JSON,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	application, err := app.New(ctx, cfg, log)
	if err != nil {
		log.Error("create whoopsync app failed", slog.Any("err", err))
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil && !isShutdownErr(err) {
		log.Error("whoopsync run failed", slog.Any("err", err))
		closeApplication(log, application, cfg.Runtime.ShutdownTimeout)
		os.Exit(1)
	}

	closeApplication(log, application, cfg.Runtime.ShutdownTimeout)

	log.Info("whoopsync stopped")
}

func closeApplication(log *slog.Logger, application *app.App, timeout time.Duration) {
	if application == nil {
		return
	}

	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	done := make(chan error, 1)

	go func() {
		done <- application.Close()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		if err != nil {
			log.Error("close whoopsync app failed", slog.Any("err", err))
		}

	case <-timer.C:
		log.Error("close whoopsync app timeout", slog.Duration("timeout", timeout))
	}
}

func isShutdownErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
