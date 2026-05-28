package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Postgres struct {
	db *sql.DB
}

func NewPostgres(db *sql.DB) (*Postgres, error) {
	if db == nil {
		return nil, errors.New("coach postgres storage: db is nil")
	}

	return &Postgres{db: db}, nil
}

func (s *Postgres) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Postgres) GetLatestSnapshot(ctx context.Context, input GetLatestSnapshotInput) (DailyHealthSnapshot, error) {
	if s == nil || s.db == nil {
		return DailyHealthSnapshot{}, errors.New("coach postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return DailyHealthSnapshot{}, errors.New("coach postgres get latest snapshot: user_id must be > 0")
	}
	if input.LookbackDays <= 0 {
		input.LookbackDays = 3
	}

	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -input.LookbackDays)

	condition := snapshotDataStateCondition(
		input.RequireReadySnapshot,
		input.AllowPartialAfterDeadline,
		input.Deadline,
		now,
	)

	query := fmt.Sprintf(`
		SELECT
			id,
			user_id,
			date,
			data_state,
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
		FROM daily_health_snapshots
		WHERE user_id = $1
		  AND date >= $2
		  AND %s
		ORDER BY date DESC, updated_at DESC, id DESC
		LIMIT 1
	`, condition)

	var snapshot DailyHealthSnapshot
	err := s.db.QueryRowContext(ctx, query, input.UserID, dateString(cutoff)).Scan(snapshotScanDest(&snapshot)...)
	if errors.Is(err, sql.ErrNoRows) {
		return DailyHealthSnapshot{}, ErrNotFound
	}
	if err != nil {
		return DailyHealthSnapshot{}, fmt.Errorf("coach postgres get latest snapshot: %w", err)
	}

	return snapshot, nil
}

func (s *Postgres) GetSnapshotForWakePlan(ctx context.Context, input GetSnapshotForWakePlanInput) (DailyHealthSnapshot, error) {
	if s == nil || s.db == nil {
		return DailyHealthSnapshot{}, errors.New("coach postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return DailyHealthSnapshot{}, errors.New("coach postgres get snapshot for wake plan: user_id must be > 0")
	}
	if input.WakePlanDate.IsZero() {
		return DailyHealthSnapshot{}, errors.New("coach postgres get snapshot for wake plan: wake_plan_date is required")
	}

	now := time.Now().UTC()
	condition := snapshotDataStateCondition(
		input.RequireReadySnapshot,
		input.AllowPartialAfterDeadline,
		input.Deadline,
		now,
	)

	query := fmt.Sprintf(`
		SELECT
			id,
			user_id,
			date,
			data_state,
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
		FROM daily_health_snapshots
		WHERE user_id = $1
		  AND date = $2
		  AND %s
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`, condition)

	var snapshot DailyHealthSnapshot
	err := s.db.QueryRowContext(ctx, query, input.UserID, dateString(input.WakePlanDate)).Scan(snapshotScanDest(&snapshot)...)
	if errors.Is(err, sql.ErrNoRows) {
		return DailyHealthSnapshot{}, ErrNotFound
	}
	if err != nil {
		return DailyHealthSnapshot{}, fmt.Errorf("coach postgres get snapshot for wake plan: %w", err)
	}

	return snapshot, nil
}

func (s *Postgres) GetActiveWakePlans(ctx context.Context, input GetActiveWakePlansInput) ([]WakePlan, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("coach postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return nil, errors.New("coach postgres get active wake plans: user_id must be > 0")
	}
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	if input.Limit <= 0 {
		input.Limit = 10
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			user_id,
			date,
			wake_at,
			prepare_at,
			final_deadline_at,
			status,
			source,
			wake_receipt_job_id,
			final_report_job_id,
			fallback_job_id,
			created_at,
			updated_at
		FROM wake_plans
		WHERE user_id = $1
		  AND status NOT IN ('cancelled', 'done', 'failed')
		  AND final_deadline_at >= $2
		ORDER BY wake_at ASC, id ASC
		LIMIT $3
	`, input.UserID, input.Now.UTC(), input.Limit)
	if err != nil {
		return nil, fmt.Errorf("coach postgres get active wake plans: %w", err)
	}
	defer rows.Close()

	out := make([]WakePlan, 0, input.Limit)

	for rows.Next() {
		var wakePlan WakePlan
		if err := rows.Scan(wakePlanScanDest(&wakePlan)...); err != nil {
			return nil, fmt.Errorf("coach postgres scan wake plan: %w", err)
		}

		out = append(out, wakePlan)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("coach postgres wake plan rows: %w", err)
	}

	return out, nil
}

func (s *Postgres) GetMorningAdvice(ctx context.Context, input GetMorningAdviceInput) (MorningAdvice, error) {
	if s == nil || s.db == nil {
		return MorningAdvice{}, errors.New("coach postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return MorningAdvice{}, errors.New("coach postgres get morning advice: user_id must be > 0")
	}
	if input.Date.IsZero() {
		return MorningAdvice{}, errors.New("coach postgres get morning advice: date is required")
	}

	promptVersion := strings.TrimSpace(input.PromptVersion)
	if promptVersion == "" {
		return MorningAdvice{}, errors.New("coach postgres get morning advice: prompt_version is required")
	}

	var advice MorningAdvice
	err := s.db.QueryRowContext(ctx, `
		SELECT
			id,
			user_id,
			date,
			snapshot_id,
			COALESCE(model, 'unknown') AS model,
			prompt_version,
			COALESCE(day_type, 'unknown') AS day_type,
			COALESCE(main_signal, '') AS main_signal,
			COALESCE(rendered_text, '') AS advice_text,
			COALESCE(motto, '') AS motto,
			COALESCE(advice_json, '{}'::jsonb) AS payload_json,
			updated_at AS generated_at,
			created_at,
			updated_at
		FROM morning_advice
		WHERE user_id = $1
		  AND date = $2
		  AND prompt_version = $3
		  AND status = 'ready'
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`, input.UserID, dateString(input.Date), promptVersion).Scan(morningAdviceScanDest(&advice)...)
	if errors.Is(err, sql.ErrNoRows) {
		return MorningAdvice{}, ErrNotFound
	}
	if err != nil {
		return MorningAdvice{}, fmt.Errorf("coach postgres get morning advice: %w", err)
	}

	return advice, nil
}

func (s *Postgres) UpsertMorningAdvice(ctx context.Context, input UpsertMorningAdviceInput) (MorningAdvice, error) {
	if s == nil || s.db == nil {
		return MorningAdvice{}, errors.New("coach postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return MorningAdvice{}, errors.New("coach postgres upsert morning advice: user_id must be > 0")
	}
	if input.Date.IsZero() {
		return MorningAdvice{}, errors.New("coach postgres upsert morning advice: date is required")
	}
	if input.SnapshotID <= 0 {
		return MorningAdvice{}, errors.New("coach postgres upsert morning advice: snapshot_id must be > 0")
	}

	input.Model = strings.TrimSpace(input.Model)
	if input.Model == "" {
		input.Model = "unknown"
	}

	input.PromptVersion = strings.TrimSpace(input.PromptVersion)
	if input.PromptVersion == "" {
		return MorningAdvice{}, errors.New("coach postgres upsert morning advice: prompt_version is required")
	}

	input.DayType = strings.TrimSpace(input.DayType)
	if input.DayType == "" {
		input.DayType = "unknown"
	}

	input.MainSignal = strings.TrimSpace(input.MainSignal)
	if input.MainSignal == "" {
		return MorningAdvice{}, errors.New("coach postgres upsert morning advice: main_signal is required")
	}

	input.AdviceText = strings.TrimSpace(input.AdviceText)
	if input.AdviceText == "" {
		return MorningAdvice{}, errors.New("coach postgres upsert morning advice: advice_text is required")
	}

	input.Motto = strings.TrimSpace(input.Motto)
	if input.Motto == "" {
		return MorningAdvice{}, errors.New("coach postgres upsert morning advice: motto is required")
	}

	payload := input.PayloadJSON
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if !json.Valid(payload) {
		return MorningAdvice{}, errors.New("coach postgres upsert morning advice: payload_json is invalid")
	}

	var advice MorningAdvice
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO morning_advice (
			user_id,
			snapshot_id,
			date,
			status,
			model,
			prompt_version,
			main_signal,
			day_type,
			advice_json,
			rendered_text,
			motto,
			error_message,
			processing_by,
			processing_until,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			'ready',
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			NULL,
			NULL,
			NULL,
			NOW(),
			NOW()
		)
		ON CONFLICT (user_id, date, prompt_version)
		DO UPDATE SET
			snapshot_id = EXCLUDED.snapshot_id,
			status = 'ready',
			model = EXCLUDED.model,
			main_signal = EXCLUDED.main_signal,
			day_type = EXCLUDED.day_type,
			advice_json = EXCLUDED.advice_json,
			rendered_text = EXCLUDED.rendered_text,
			motto = EXCLUDED.motto,
			error_message = NULL,
			processing_by = NULL,
			processing_until = NULL,
			updated_at = NOW()
		RETURNING
			id,
			user_id,
			date,
			snapshot_id,
			COALESCE(model, 'unknown') AS model,
			prompt_version,
			COALESCE(day_type, 'unknown') AS day_type,
			COALESCE(main_signal, '') AS main_signal,
			COALESCE(rendered_text, '') AS advice_text,
			COALESCE(motto, '') AS motto,
			COALESCE(advice_json, '{}'::jsonb) AS payload_json,
			updated_at AS generated_at,
			created_at,
			updated_at
	`,
		input.UserID,
		input.SnapshotID,
		dateString(input.Date),
		input.Model,
		input.PromptVersion,
		input.MainSignal,
		input.DayType,
		[]byte(payload),
		input.AdviceText,
		input.Motto,
	).Scan(morningAdviceScanDest(&advice)...)
	if err != nil {
		return MorningAdvice{}, fmt.Errorf("coach postgres upsert morning advice: %w", err)
	}

	return advice, nil
}

func snapshotDataStateCondition(requireReady bool, allowPartialAfterDeadline bool, deadline time.Time, now time.Time) string {
	if !requireReady {
		return "data_state IN ('ready', 'partial')"
	}

	if allowPartialAfterDeadline && !deadline.IsZero() && !now.Before(deadline.UTC()) {
		return "data_state IN ('ready', 'partial')"
	}

	return "data_state = 'ready'"
}

func snapshotScanDest(s *DailyHealthSnapshot) []any {
	return []any{
		&s.ID,
		&s.UserID,
		&s.Date,
		&s.DataState,
		&s.SleepScore,
		&s.RecoveryScore,
		&s.DayStrain,
		&s.SleepMinutes,
		&s.SleepNeededMinutes,
		&s.SleepVsNeedPct,
		&s.AwakeMinutes,
		&s.LightSleepMinutes,
		&s.DeepSleepMinutes,
		&s.REMSleepMinutes,
		&s.RestorativeMinutes,
		&s.SleepEfficiencyPct,
		&s.SleepConsistencyPct,
		&s.RespiratoryRate,
		&s.HRVRMSSDMS,
		&s.RestingHeartRateBPM,
		&s.SpO2Pct,
		&s.SkinTempCelsius,
		&s.SourceUpdatedAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	}
}

func wakePlanScanDest(w *WakePlan) []any {
	return []any{
		&w.ID,
		&w.UserID,
		&w.Date,
		&w.WakeAt,
		&w.PrepareAt,
		&w.FinalDeadlineAt,
		&w.Status,
		&w.Source,
		&w.WakeReceiptJobID,
		&w.FinalReportJobID,
		&w.FallbackJobID,
		&w.CreatedAt,
		&w.UpdatedAt,
	}
}

func morningAdviceScanDest(a *MorningAdvice) []any {
	return []any{
		&a.ID,
		&a.UserID,
		&a.Date,
		&a.SnapshotID,
		&a.Model,
		&a.PromptVersion,
		&a.DayType,
		&a.MainSignal,
		&a.AdviceText,
		&a.Motto,
		&a.PayloadJSON,
		&a.GeneratedAt,
		&a.CreatedAt,
		&a.UpdatedAt,
	}
}

func dateString(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}
