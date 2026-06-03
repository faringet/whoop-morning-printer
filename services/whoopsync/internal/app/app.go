package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/internal/platform/postgres"
	svcconfig "github.com/faringet/whoop-morning-printer/services/whoopsync/config"
	"github.com/faringet/whoop-morning-printer/services/whoopsync/internal/storage"
	"github.com/faringet/whoop-morning-printer/services/whoopsync/internal/syncer"
	"github.com/faringet/whoop-morning-printer/services/whoopsync/internal/whoopapi"
)

type App struct {
	cfg *svcconfig.Config
	log *slog.Logger

	db     *sql.DB
	store  storage.Store
	oauth  *whoopapi.OAuthClient
	api    *whoopapi.Client
	worker *syncer.Worker
}

func New(ctx context.Context, cfg *svcconfig.Config, log *slog.Logger) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("whoopsync app: config is nil")
	}
	if log == nil {
		log = slog.Default()
	}

	oauthClient, err := whoopapi.NewOAuthClient(whoopapi.OAuthConfig{
		ClientID:     cfg.Whoop.ClientID,
		ClientSecret: cfg.Whoop.ClientSecret,
		RedirectURL:  cfg.Whoop.RedirectURL,
		Scopes:       cfg.Whoop.Scopes,
		AuthURL:      cfg.Whoop.AuthURL,
		TokenURL:     cfg.Whoop.TokenURL,
		HTTPTimeout:  cfg.Whoop.HTTPTimeout,
	})
	if err != nil {
		return nil, err
	}

	app := &App{
		cfg:   cfg,
		log:   log,
		oauth: oauthClient,
	}

	if isOAuthURLMode(cfg.WhoopSync.Mode) {
		return app, nil
	}

	db, err := postgres.Open(postgres.Config{
		DSN:             cfg.Storage.Postgres.DSN,
		MaxOpenConns:    cfg.Storage.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Storage.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Storage.Postgres.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Storage.Postgres.ConnMaxIdleTime,
	})
	if err != nil {
		return nil, err
	}

	waitCtx, cancel := context.WithTimeout(ctx, cfg.WhoopSync.StartupTimeout)
	defer cancel()

	if err := postgres.WaitReady(waitCtx, db, cfg.WhoopSync.PingInterval); err != nil {
		_ = db.Close()
		return nil, err
	}

	store, err := storage.NewPostgresStore(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	apiClient, err := whoopapi.NewClient(whoopapi.ClientConfig{
		BaseURL:     cfg.Whoop.APIBaseURL,
		HTTPTimeout: cfg.Whoop.HTTPTimeout,
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	worker, err := syncer.NewWorker(syncer.WorkerConfig{
		UserID:           cfg.WhoopSync.UserID,
		LookbackDays:     cfg.WhoopSync.LookbackDays,
		TokenRefreshSkew: cfg.WhoopSync.TokenRefreshSkew,
		Interval:         cfg.WhoopSync.Interval,
	}, store, oauthClient, apiClient, log)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	app.db = db
	app.store = store
	app.api = apiClient
	app.worker = worker

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	if a == nil {
		return fmt.Errorf("whoopsync app: app is nil")
	}

	mode := strings.ToLower(strings.TrimSpace(a.cfg.WhoopSync.Mode))

	switch mode {
	case "oauth_url":
		return a.runOAuthURLMode()

	case "oauth_code":
		if a.worker == nil {
			return fmt.Errorf("whoopsync app: worker is nil")
		}
		return a.worker.ExchangeCode(ctx, a.cfg.WhoopSync.AuthorizationCode)

	case "once":
		if a.worker == nil {
			return fmt.Errorf("whoopsync app: worker is nil")
		}
		return a.worker.SyncOnce(ctx)

	case "interval":
		if a.worker == nil {
			return fmt.Errorf("whoopsync app: worker is nil")
		}
		return a.worker.RunInterval(ctx)

	case "wake_watch":
		if a.worker == nil {
			return fmt.Errorf("whoopsync app: worker is nil")
		}
		return a.worker.RunWakeWatch(ctx)

	default:
		return fmt.Errorf("whoopsync app: unsupported mode %q", mode)
	}
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}

	if a.store != nil {
		return a.store.Close()
	}

	if a.db != nil {
		return a.db.Close()
	}

	return nil
}

func (a *App) runOAuthURLMode() error {
	state, err := randomState(16)
	if err != nil {
		a.log.Warn("generate oauth state failed, using timestamp fallback", slog.Any("err", err))
		state = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	authURL, err := a.oauth.AuthorizationURL(state)
	if err != nil {
		return err
	}

	a.log.Info("open WHOOP authorization URL in browser",
		slog.String("url", authURL),
		slog.String("state", state),
	)

	return nil
}

func isOAuthURLMode(mode string) bool {
	return strings.EqualFold(strings.TrimSpace(mode), "oauth_url")
}

func randomState(size int) (string, error) {
	if size <= 0 {
		size = 16
	}

	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
