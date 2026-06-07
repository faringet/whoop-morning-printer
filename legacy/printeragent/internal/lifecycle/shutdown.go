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
)

type App interface {
	Run(ctx context.Context) error
}

type Logger interface {
	Info(msg string, kv ...interface{})
	Warn(msg string, kv ...interface{})
	Error(msg string, kv ...interface{})
}

type Options struct {
	ShutdownTimeout time.Duration

	Mode       string
	OutputMode string
	GatewayURL string
	LogFile    string

	Out io.Writer
}

func RunWithGracefulShutdown(app App, log Logger, opts Options) error {
	if app == nil {
		return errors.New("printeragent lifecycle: app is nil")
	}
	if log == nil {
		return errors.New("printeragent lifecycle: logger is nil")
	}

	if opts.ShutdownTimeout <= 0 {
		opts.ShutdownTimeout = 15 * time.Second
	}

	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	signalCh := make(chan os.Signal, 2)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(signalCh)

	appDone := make(chan error, 1)

	go func() {
		appDone <- app.Run(runCtx)
	}()

	select {
	case err := <-appDone:
		if err != nil && !isShutdownErr(err) {
			return err
		}

		return nil

	case sig := <-signalCh:
		return shutdownAfterSignal(sig, signalCh, appDone, cancelRun, log, opts)
	}
}

func shutdownAfterSignal(
	sig os.Signal,
	signalCh <-chan os.Signal,
	appDone <-chan error,
	cancelRun context.CancelFunc,
	log Logger,
	opts Options,
) error {
	log.Info("shutdown signal received",
		"signal", sig.String(),
		"shutdown_timeout", opts.ShutdownTimeout,
	)

	PrintGracefulShutdownBanner(opts.Out, ShutdownBannerInfo{
		Signal:    sig.String(),
		Timeout:   opts.ShutdownTimeout,
		Mode:      opts.Mode,
		Output:    opts.OutputMode,
		Gateway:   opts.GatewayURL,
		LogFile:   opts.LogFile,
		Timestamp: time.Now().UTC(),
	})

	cancelRun()

	timer := time.NewTimer(opts.ShutdownTimeout)
	defer timer.Stop()

	select {
	case err := <-appDone:
		if err != nil && !isShutdownErr(err) {
			return err
		}

		log.Info("graceful shutdown completed",
			"signal", sig.String(),
		)

		PrintShutdownCompleteBanner(opts.Out)

		return nil

	case secondSig := <-signalCh:
		log.Warn("second shutdown signal received, forcing exit",
			"first_signal", sig.String(),
			"second_signal", secondSig.String(),
		)

		return fmt.Errorf("forced shutdown after second signal %s", secondSig.String())

	case <-timer.C:
		log.Error("graceful shutdown timeout exceeded",
			"timeout", opts.ShutdownTimeout,
		)

		return fmt.Errorf("graceful shutdown timeout exceeded after %s: %w", opts.ShutdownTimeout, context.DeadlineExceeded)
	}
}

func isShutdownErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
