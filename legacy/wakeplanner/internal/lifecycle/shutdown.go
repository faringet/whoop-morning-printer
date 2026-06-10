package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/logger"
)

type App interface {
	Run(context.Context) error
	Close() error
}

type Options struct {
	ShutdownTimeout time.Duration

	DryRun             bool
	SleepAfterPlanning bool
	GatewayURL         string
	LogFile            string

	Out io.Writer
}

func RunWithGracefulShutdown(app App, log *logger.Logger, opts Options) error {
	if app == nil {
		return errors.New("wakeplanner lifecycle: app is nil")
	}

	if opts.ShutdownTimeout <= 0 {
		opts.ShutdownTimeout = 15 * time.Second
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	errCh := make(chan error, 1)

	go func() {
		errCh <- app.Run(ctx)
	}()

	select {
	case err := <-errCh:
		if err != nil && !isShutdownErr(err) {
			return err
		}

		return nil

	case <-ctx.Done():
		if log != nil {
			log.Info("shutdown signal received",
				"reason", ctx.Err(),
				"shutdown_timeout", opts.ShutdownTimeout,
			)
		}

		printShutdownBanner(opts.Out, opts)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), opts.ShutdownTimeout)
		defer cancel()

		if err := app.Close(); err != nil && log != nil {
			log.Error("app close failed", "err", err)
		}

		select {
		case err := <-errCh:
			if err != nil && !isShutdownErr(err) {
				return err
			}

			if log != nil {
				log.Info("graceful shutdown completed")
			}

			return nil

		case <-shutdownCtx.Done():
			return fmt.Errorf("wakeplanner lifecycle: shutdown timeout: %w", shutdownCtx.Err())
		}
	}
}

func isShutdownErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func printShutdownBanner(w io.Writer, opts Options) {
	if w == nil {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "----------------------------------------")
	fmt.Fprintln(w, "  WAKEPLANNER SHUTDOWN")
	fmt.Fprintln(w, "----------------------------------------")
	fmt.Fprintf(w, "  dry_run             %s\n", yesNo(opts.DryRun))
	fmt.Fprintf(w, "  sleep_after_plan    %s\n", yesNo(opts.SleepAfterPlanning))
	fmt.Fprintf(w, "  gateway             %s\n", valueOrDash(opts.GatewayURL))
	fmt.Fprintf(w, "  logs                %s\n", valueOrDash(opts.LogFile))
	fmt.Fprintln(w, "----------------------------------------")
	fmt.Fprintln(w)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}

	return value
}
