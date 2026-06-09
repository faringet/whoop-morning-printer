package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	platformpg "github.com/faringet/whoop-morning-printer/internal/platform/postgres"
	"github.com/faringet/whoop-morning-printer/services/printergateway/config"
	"github.com/faringet/whoop-morning-printer/services/printergateway/internal/httpapi"
	"github.com/faringet/whoop-morning-printer/services/printergateway/internal/storage"
)

type App struct {
	cfg *config.Config
	log *slog.Logger

	store  storage.Store
	server *http.Server
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	if cfg == nil {
		return nil, errors.New("printergateway app: config is nil")
	}
	if log == nil {
		log = slog.Default()
	}

	appLog := log.With(
		slog.String("layer", "app"),
		slog.String("module", "printergateway.app"),
	)

	st, err := openStore(cfg, appLog)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	authToken, err := cfg.PrinterGateway.AuthTokenValue()
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("load auth token: %w", err)
	}

	apiServer, err := httpapi.NewServer(log, st, authToken)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create http api server: %w", err)
	}

	server := &http.Server{
		Addr:              cfg.PrinterGateway.HTTPAddr,
		Handler:           apiServer,
		ReadHeaderTimeout: cfg.PrinterGateway.ReadHeaderTimeout,
		ReadTimeout:       cfg.PrinterGateway.ReadTimeout,
		WriteTimeout:      cfg.PrinterGateway.WriteTimeout,
		IdleTimeout:       cfg.PrinterGateway.IdleTimeout,
	}

	appLog.Info("printergateway initialized",
		slog.String("http_addr", cfg.PrinterGateway.HTTPAddr),
		slog.Duration("read_header_timeout", cfg.PrinterGateway.ReadHeaderTimeout),
		slog.Duration("read_timeout", cfg.PrinterGateway.ReadTimeout),
		slog.Duration("write_timeout", cfg.PrinterGateway.WriteTimeout),
		slog.Duration("idle_timeout", cfg.PrinterGateway.IdleTimeout),
		slog.Int("postgres_max_open_conns", cfg.Storage.Postgres.MaxOpenConns),
		slog.Int("postgres_max_idle_conns", cfg.Storage.Postgres.MaxIdleConns),
	)

	return &App{
		cfg:    cfg,
		log:    appLog,
		store:  st,
		server: server,
	}, nil
}

func openStore(cfg *config.Config, log *slog.Logger) (storage.Store, error) {
	if cfg == nil {
		return nil, errors.New("printergateway app: config is nil")
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

	if err := waitForPostgres(context.Background(), db, cfg.PrinterGateway.StartupTimeout, cfg.PrinterGateway.PingInterval, log); err != nil {
		_ = db.Close()
		return nil, err
	}

	st, err := storage.NewPostgres(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create postgres storage: %w", err)
	}

	return st, nil
}

func waitForPostgres(
	ctx context.Context,
	pinger interface {
		PingContext(context.Context) error
	},
	timeout time.Duration,
	interval time.Duration,
	log *slog.Logger,
) error {
	if pinger == nil {
		return errors.New("printergateway app: postgres pinger is nil")
	}

	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastErr error

	for {
		pingCtx, pingCancel := context.WithTimeout(ctx, interval)
		err := pinger.PingContext(pingCtx)
		pingCancel()

		if err == nil {
			if log != nil {
				log.Info("postgres is ready")
			}
			return nil
		}

		lastErr = err

		if log != nil {
			log.Warn("postgres is not ready yet",
				slog.Any("err", err),
				slog.Duration("retry_in", interval),
			)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for postgres: %w: last error: %v", ctx.Err(), lastErr)

		case <-ticker.C:
		}
	}
}

func (a *App) Run(ctx context.Context) error {
	if a == nil {
		return errors.New("printergateway app: app is nil")
	}
	if a.server == nil {
		return errors.New("printergateway app: http server is nil")
	}

	a.log.Info("http server starting",
		slog.String("addr", a.server.Addr),
	)

	errCh := make(chan error, 1)

	go func() {
		err := a.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}

		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http server failed: %w", err)
		}

		return nil

	case <-ctx.Done():
		timeout := a.cfg.Runtime.ShutdownTimeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}

		a.log.Info("http server shutdown started",
			slog.Duration("timeout", timeout),
			slog.Any("reason", ctx.Err()),
		)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http server shutdown: %w", err)
		}

		err := <-errCh
		if err != nil {
			return fmt.Errorf("http server stopped with error: %w", err)
		}

		a.log.Info("http server stopped")

		return ctx.Err()
	}
}

func (a *App) Close() error {
	if a == nil || a.store == nil {
		return nil
	}

	return a.store.Close()
}
