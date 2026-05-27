package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/whoopsync/internal/whoopapi"
)

var ErrNotFound = errors.New("storage: not found")

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) (*PostgresStore, error) {
	if db == nil {
		return nil, fmt.Errorf("storage: db is nil")
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *PostgresStore) EnsureUser(ctx context.Context, userID int64, timezone string) error {
	if userID <= 0 {
		return fmt.Errorf("storage: user_id must be > 0")
	}

	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		timezone = "Europe/Moscow"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storage: begin ensure user tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (id, timezone, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (id)
		DO UPDATE SET
			timezone = EXCLUDED.timezone,
			updated_at = NOW()
	`, userID, timezone); err != nil {
		return fmt.Errorf("storage: ensure user: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		SELECT setval(
			pg_get_serial_sequence('users', 'id'),
			GREATEST((SELECT COALESCE(MAX(id), 1) FROM users), 1),
			true
		)
	`); err != nil {
		return fmt.Errorf("storage: sync users id sequence: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("storage: commit ensure user tx: %w", err)
	}

	return nil
}

// todo добавить шифрование
func (s *PostgresStore) SaveTokens(ctx context.Context, tokens Tokens) error {
	if tokens.UserID <= 0 {
		return fmt.Errorf("storage: tokens.user_id must be > 0")
	}

	tokens.AccessToken = strings.TrimSpace(tokens.AccessToken)
	tokens.RefreshToken = strings.TrimSpace(tokens.RefreshToken)
	tokens.TokenType = strings.TrimSpace(tokens.TokenType)

	if tokens.AccessToken == "" {
		return fmt.Errorf("storage: tokens.access_token is required")
	}
	if tokens.RefreshToken == "" {
		return fmt.Errorf("storage: tokens.refresh_token is required")
	}
	if tokens.TokenType == "" {
		tokens.TokenType = "Bearer"
	}
	if tokens.ExpiresAt.IsZero() {
		return fmt.Errorf("storage: tokens.expires_at is required")
	}

	scopesCSV := strings.Join(cleanScopes(tokens.Scopes), ",")

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whoop_tokens (
			user_id,
			access_token_encrypted,
			refresh_token_encrypted,
			token_type,
			scopes,
			expires_at,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			CASE
				WHEN $5 = '' THEN '{}'::text[]
				ELSE string_to_array($5, ',')
			END,
			$6,
			NOW(),
			NOW()
		)
		ON CONFLICT (user_id)
		DO UPDATE SET
			access_token_encrypted = EXCLUDED.access_token_encrypted,
			refresh_token_encrypted = EXCLUDED.refresh_token_encrypted,
			token_type = EXCLUDED.token_type,
			scopes = EXCLUDED.scopes,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
	`, tokens.UserID, tokens.AccessToken, tokens.RefreshToken, tokens.TokenType, scopesCSV, tokens.ExpiresAt.UTC())
	if err != nil {
		return fmt.Errorf("storage: save tokens: %w", err)
	}

	return nil
}

// todo добавить шифрование
func (s *PostgresStore) GetTokens(ctx context.Context, userID int64) (Tokens, error) {
	if userID <= 0 {
		return Tokens{}, fmt.Errorf("storage: user_id must be > 0")
	}

	var tokens Tokens
	var scopesCSV string

	err := s.db.QueryRowContext(ctx, `
		SELECT
			user_id,
			access_token_encrypted,
			refresh_token_encrypted,
			token_type,
			array_to_string(scopes, ','),
			expires_at,
			created_at,
			updated_at
		FROM whoop_tokens
		WHERE user_id = $1
	`, userID).Scan(
		&tokens.UserID,
		&tokens.AccessToken,
		&tokens.RefreshToken,
		&tokens.TokenType,
		&scopesCSV,
		&tokens.ExpiresAt,
		&tokens.CreatedAt,
		&tokens.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Tokens{}, ErrNotFound
	}
	if err != nil {
		return Tokens{}, fmt.Errorf("storage: get tokens: %w", err)
	}

	tokens.Scopes = splitScopes(scopesCSV)

	return tokens, nil
}

func (s *PostgresStore) UpsertRawWHOOPObject(ctx context.Context, object RawWHOOPObject) error {
	if object.UserID <= 0 {
		return fmt.Errorf("storage: raw object user_id must be > 0")
	}
	if strings.TrimSpace(string(object.ObjectType)) == "" {
		return fmt.Errorf("storage: raw object object_type is required")
	}
	if strings.TrimSpace(object.WHOOPID) == "" {
		return fmt.Errorf("storage: raw object whoop_id is required")
	}
	if len(object.PayloadJSON) == 0 {
		return fmt.Errorf("storage: raw object payload_json is required")
	}
	if !json.Valid(object.PayloadJSON) {
		return fmt.Errorf("storage: raw object payload_json is invalid")
	}
	if object.FetchedAt.IsZero() {
		object.FetchedAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO raw_whoop_objects (
			user_id,
			object_type,
			whoop_id,
			start_at,
			end_at,
			score_state,
			payload_json,
			fetched_at,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7::jsonb,
			$8,
			NOW(),
			NOW()
		)
		ON CONFLICT (user_id, object_type, whoop_id)
		DO UPDATE SET
			start_at = EXCLUDED.start_at,
			end_at = EXCLUDED.end_at,
			score_state = EXCLUDED.score_state,
			payload_json = EXCLUDED.payload_json,
			fetched_at = EXCLUDED.fetched_at,
			updated_at = NOW()
	`,
		object.UserID,
		string(object.ObjectType),
		object.WHOOPID,
		timePtrValue(object.StartAt),
		timePtrValue(object.EndAt),
		scoreStatePtrValue(object.ScoreState),
		string(object.PayloadJSON),
		object.FetchedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("storage: upsert raw whoop object: %w", err)
	}

	return nil
}

func (s *PostgresStore) UpsertDailyHealthSnapshot(ctx context.Context, snapshot DailyHealthSnapshot) error {
	if snapshot.UserID <= 0 {
		return fmt.Errorf("storage: snapshot user_id must be > 0")
	}
	if snapshot.Date.IsZero() {
		return fmt.Errorf("storage: snapshot date is required")
	}
	if snapshot.DataState == "" {
		snapshot.DataState = DataStatePending
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO daily_health_snapshots (
			user_id,
			date,
			data_state,
			sleep_whoop_id,
			recovery_whoop_id,
			cycle_whoop_id,
			sleep_score,
			recovery_score,
			day_strain,
			sleep_minutes,
			sleep_needed_minutes,
			sleep_vs_need_pct,
			awake_minutes,
			light_sleep_minutes,
			deep_sleep_minutes,
			rem_sleep_minutes,
			restorative_sleep_minutes,
			sleep_efficiency_pct,
			sleep_consistency_pct,
			respiratory_rate,
			hrv_rmssd_ms,
			resting_heart_rate_bpm,
			spo2_pct,
			skin_temp_celsius,
			source_updated_at,
			created_at,
			updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24,
			$25, NOW(), NOW()
		)
		ON CONFLICT (user_id, date)
		DO UPDATE SET
			data_state = EXCLUDED.data_state,
			sleep_whoop_id = EXCLUDED.sleep_whoop_id,
			recovery_whoop_id = EXCLUDED.recovery_whoop_id,
			cycle_whoop_id = EXCLUDED.cycle_whoop_id,
			sleep_score = EXCLUDED.sleep_score,
			recovery_score = EXCLUDED.recovery_score,
			day_strain = EXCLUDED.day_strain,
			sleep_minutes = EXCLUDED.sleep_minutes,
			sleep_needed_minutes = EXCLUDED.sleep_needed_minutes,
			sleep_vs_need_pct = EXCLUDED.sleep_vs_need_pct,
			awake_minutes = EXCLUDED.awake_minutes,
			light_sleep_minutes = EXCLUDED.light_sleep_minutes,
			deep_sleep_minutes = EXCLUDED.deep_sleep_minutes,
			rem_sleep_minutes = EXCLUDED.rem_sleep_minutes,
			restorative_sleep_minutes = EXCLUDED.restorative_sleep_minutes,
			sleep_efficiency_pct = EXCLUDED.sleep_efficiency_pct,
			sleep_consistency_pct = EXCLUDED.sleep_consistency_pct,
			respiratory_rate = EXCLUDED.respiratory_rate,
			hrv_rmssd_ms = EXCLUDED.hrv_rmssd_ms,
			resting_heart_rate_bpm = EXCLUDED.resting_heart_rate_bpm,
			spo2_pct = EXCLUDED.spo2_pct,
			skin_temp_celsius = EXCLUDED.skin_temp_celsius,
			source_updated_at = EXCLUDED.source_updated_at,
			updated_at = NOW()
	`,
		snapshot.UserID,
		dateString(snapshot.Date),
		string(snapshot.DataState),
		stringPtrValue(snapshot.SleepWHOOPID),
		stringPtrValue(snapshot.RecoveryWHOOPID),
		stringPtrValue(snapshot.CycleWHOOPID),
		intPtrValue(snapshot.SleepScore),
		intPtrValue(snapshot.RecoveryScore),
		floatPtrValue(snapshot.DayStrain),
		intPtrValue(snapshot.SleepMinutes),
		intPtrValue(snapshot.SleepNeededMinutes),
		intPtrValue(snapshot.SleepVsNeedPct),
		intPtrValue(snapshot.AwakeMinutes),
		intPtrValue(snapshot.LightSleepMinutes),
		intPtrValue(snapshot.DeepSleepMinutes),
		intPtrValue(snapshot.REMSleepMinutes),
		intPtrValue(snapshot.RestorativeMinutes),
		floatPtrValue(snapshot.SleepEfficiencyPct),
		floatPtrValue(snapshot.SleepConsistencyPct),
		floatPtrValue(snapshot.RespiratoryRate),
		floatPtrValue(snapshot.HRVRMSSDMS),
		intPtrValue(snapshot.RestingHeartRateBPM),
		floatPtrValue(snapshot.SpO2Pct),
		floatPtrValue(snapshot.SkinTempCelsius),
		timePtrValue(snapshot.SourceUpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("storage: upsert daily health snapshot: %w", err)
	}

	return nil
}

func cleanScopes(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))

	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}

		seen[scope] = struct{}{}
		out = append(out, scope)
	}

	return out
}

func splitScopes(scopesCSV string) []string {
	if strings.TrimSpace(scopesCSV) == "" {
		return nil
	}

	return cleanScopes(strings.Split(scopesCSV, ","))
}

func dateString(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

func stringPtrValue(v *string) any {
	if v == nil {
		return nil
	}

	value := strings.TrimSpace(*v)
	if value == "" {
		return nil
	}

	return value
}

func intPtrValue(v *int) any {
	if v == nil {
		return nil
	}

	return *v
}

func floatPtrValue(v *float64) any {
	if v == nil {
		return nil
	}

	return *v
}

func timePtrValue(v *time.Time) any {
	if v == nil || v.IsZero() {
		return nil
	}

	return v.UTC()
}

func scoreStatePtrValue(v *whoopapi.ScoreState) any {
	if v == nil {
		return nil
	}

	value := strings.TrimSpace(string(*v))
	if value == "" {
		return nil
	}

	return value
}
