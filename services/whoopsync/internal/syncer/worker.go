package syncer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/whoopsync/internal/storage"
	"github.com/faringet/whoop-morning-printer/services/whoopsync/internal/whoopapi"
)

type WorkerConfig struct {
	UserID int64

	LookbackDays     int
	TokenRefreshSkew time.Duration
	Interval         time.Duration
	Timezone         string
}

type Worker struct {
	cfg   WorkerConfig
	store storage.Store
	oauth *whoopapi.OAuthClient
	api   *whoopapi.Client
	log   *slog.Logger
	now   func() time.Time
}

func NewWorker(cfg WorkerConfig, store storage.Store, oauth *whoopapi.OAuthClient, api *whoopapi.Client, log *slog.Logger) (*Worker, error) {
	if cfg.UserID <= 0 {
		return nil, fmt.Errorf("syncer: user_id must be > 0")
	}
	if cfg.LookbackDays <= 0 {
		cfg.LookbackDays = 3
	}
	if cfg.TokenRefreshSkew <= 0 {
		cfg.TokenRefreshSkew = 2 * time.Minute
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Minute
	}
	if strings.TrimSpace(cfg.Timezone) == "" {
		cfg.Timezone = defaultTimezone
	}
	if store == nil {
		return nil, fmt.Errorf("syncer: store is nil")
	}
	if oauth == nil {
		return nil, fmt.Errorf("syncer: oauth client is nil")
	}
	if api == nil {
		return nil, fmt.Errorf("syncer: whoop api client is nil")
	}
	if log == nil {
		log = slog.Default()
	}

	return &Worker{
		cfg:   cfg,
		store: store,
		oauth: oauth,
		api:   api,
		log:   log,
		now:   time.Now,
	}, nil
}

func (w *Worker) AuthorizationURL(state string) (string, error) {
	if w == nil {
		return "", fmt.Errorf("syncer: worker is nil")
	}

	return w.oauth.AuthorizationURL(state)
}

func (w *Worker) ExchangeCode(ctx context.Context, code string) error {
	if w == nil {
		return fmt.Errorf("syncer: worker is nil")
	}

	if err := w.store.EnsureUser(ctx, w.cfg.UserID, w.cfg.Timezone); err != nil {
		return err
	}

	token, err := w.oauth.ExchangeCode(ctx, code)
	if err != nil {
		return fmt.Errorf("syncer: exchange whoop authorization code: %w", err)
	}

	if err := w.store.SaveTokens(ctx, storage.Tokens{
		UserID:       w.cfg.UserID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Scopes:       scopesFromTokenResponse(token, nil),
		ExpiresAt:    token.ExpiresAt,
	}); err != nil {
		return err
	}

	w.log.Info("whoop tokens saved after oauth code exchange",
		slog.Int64("user_id", w.cfg.UserID),
		slog.Time("expires_at", token.ExpiresAt),
	)

	return nil
}

func (w *Worker) RunInterval(ctx context.Context) error {
	if w == nil {
		return fmt.Errorf("syncer: worker is nil")
	}

	if err := w.SyncOnce(ctx); err != nil {
		w.log.Error("initial whoop sync failed", slog.Any("err", err))
	}

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if err := w.SyncOnce(ctx); err != nil {
				w.log.Error("periodic whoop sync failed", slog.Any("err", err))
				continue
			}
		}
	}
}

func (w *Worker) SyncOnce(ctx context.Context) error {
	if w == nil {
		return fmt.Errorf("syncer: worker is nil")
	}

	startedAt := w.now().UTC()

	if err := w.store.EnsureUser(ctx, w.cfg.UserID, w.cfg.Timezone); err != nil {
		return err
	}

	tokens, err := w.getFreshTokens(ctx)
	if err != nil {
		return err
	}

	end := w.now().UTC()
	start := end.AddDate(0, 0, -w.cfg.LookbackDays)

	w.log.Info("syncing whoop data",
		slog.Int64("user_id", w.cfg.UserID),
		slog.Time("start", start),
		slog.Time("end", end),
		slog.Int("lookback_days", w.cfg.LookbackDays),
	)

	sleeps, err := w.api.GetSleeps(ctx, tokens.AccessToken, whoopapi.SleepQuery{
		Start: start,
		End:   end,
		Limit: 25,
	})
	if err != nil {
		return fmt.Errorf("syncer: fetch sleeps: %w", err)
	}

	recoveries, err := w.api.GetRecoveries(ctx, tokens.AccessToken, whoopapi.RecoveryQuery{
		Start: start,
		End:   end,
		Limit: 25,
	})
	if err != nil {
		return fmt.Errorf("syncer: fetch recoveries: %w", err)
	}

	cycles, err := w.api.GetCycles(ctx, tokens.AccessToken, whoopapi.CycleQuery{
		Start: start,
		End:   end,
		Limit: 25,
	})
	if err != nil {
		return fmt.Errorf("syncer: fetch cycles: %w", err)
	}

	workouts, err := w.api.GetWorkouts(ctx, tokens.AccessToken, whoopapi.WorkoutQuery{
		Start: start,
		End:   end,
		Limit: 25,
	})
	if err != nil {
		return fmt.Errorf("syncer: fetch workouts: %w", err)
	}

	result, err := BuildSnapshot(SnapshotBuildInput{
		UserID:     w.cfg.UserID,
		Now:        end,
		Timezone:   w.cfg.Timezone,
		Sleeps:     sleeps,
		Recoveries: recoveries,
		Cycles:     cycles,
		Workouts:   workouts,
	})
	if err != nil {
		return err
	}

	for _, object := range result.RawObjects {
		if err := w.store.UpsertRawWHOOPObject(ctx, object); err != nil {
			return err
		}
	}

	if err := w.store.UpsertDailyHealthSnapshot(ctx, result.Snapshot); err != nil {
		return err
	}

	w.log.Info("whoop sync completed",
		slog.Int64("user_id", w.cfg.UserID),
		slog.String("data_state", string(result.Snapshot.DataState)),
		slog.Time("snapshot_date", result.Snapshot.Date),
		slog.Int("sleeps", len(sleeps)),
		slog.Int("recoveries", len(recoveries)),
		slog.Int("cycles", len(cycles)),
		slog.Int("workouts", len(workouts)),
		slog.Int("raw_objects", len(result.RawObjects)),
		slog.Duration("elapsed", time.Since(startedAt)),
	)

	return nil
}

func (w *Worker) getFreshTokens(ctx context.Context) (storage.Tokens, error) {
	tokens, err := w.store.GetTokens(ctx, w.cfg.UserID)
	if errors.Is(err, storage.ErrNotFound) {
		return storage.Tokens{}, fmt.Errorf("syncer: whoop tokens not found for user_id=%d; run oauth_url and oauth_code first", w.cfg.UserID)
	}
	if err != nil {
		return storage.Tokens{}, err
	}

	if strings.TrimSpace(tokens.AccessToken) == "" {
		return storage.Tokens{}, fmt.Errorf("syncer: stored access token is empty")
	}
	if strings.TrimSpace(tokens.RefreshToken) == "" {
		return storage.Tokens{}, fmt.Errorf("syncer: stored refresh token is empty")
	}

	refreshAfter := w.now().UTC().Add(w.cfg.TokenRefreshSkew)
	if tokens.ExpiresAt.After(refreshAfter) {
		return tokens, nil
	}

	w.log.Info("refreshing whoop access token",
		slog.Int64("user_id", w.cfg.UserID),
		slog.Time("expires_at", tokens.ExpiresAt),
		slog.Duration("refresh_skew", w.cfg.TokenRefreshSkew),
	)

	refreshed, err := w.oauth.RefreshToken(ctx, tokens.RefreshToken)
	if err != nil {
		return storage.Tokens{}, fmt.Errorf("syncer: refresh whoop token: %w", err)
	}

	freshTokens := storage.Tokens{
		UserID:       w.cfg.UserID,
		AccessToken:  refreshed.AccessToken,
		RefreshToken: refreshed.RefreshToken,
		TokenType:    refreshed.TokenType,
		Scopes:       scopesFromTokenResponse(refreshed, tokens.Scopes),
		ExpiresAt:    refreshed.ExpiresAt,
	}

	if err := w.store.SaveTokens(ctx, freshTokens); err != nil {
		return storage.Tokens{}, err
	}

	return freshTokens, nil
}

func scopesFromTokenResponse(token whoopapi.TokenResponse, fallback []string) []string {
	scopes := strings.Fields(strings.TrimSpace(token.Scope))
	if len(scopes) == 0 {
		return fallback
	}

	return scopes
}
