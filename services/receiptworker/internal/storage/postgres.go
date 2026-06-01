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
		return nil, errors.New("receiptworker postgres storage: db is nil")
	}

	return &Postgres{db: db}, nil
}

func (s *Postgres) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Postgres) GetPendingWakeReceiptTasks(ctx context.Context, input GetPendingTasksInput) ([]WakeReceiptTask, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("receiptworker postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return nil, errors.New("receiptworker get pending wake receipt tasks: user_id must be > 0")
	}
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	if input.Limit <= 0 {
		input.Limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			pj.id,
			pj.user_id,
			pj.wake_plan_id,
			pj.type,
			pj.status,
			pj.not_before,
			COALESCE(pj.payload_type, 'text/plain') AS payload_type,
			COALESCE(pj.payload_text, '') AS payload_text,
			pj.error_message,
			pj.created_at,
			pj.updated_at,

			wp.id,
			wp.user_id,
			wp.date,
			wp.wake_at,
			wp.prepare_at,
			wp.final_deadline_at,
			wp.status,
			wp.source,
			wp.wake_receipt_job_id,
			wp.final_report_job_id,
			wp.fallback_job_id,
			wp.created_at,
			wp.updated_at
		FROM print_jobs pj
		JOIN wake_plans wp ON wp.id = pj.wake_plan_id
		WHERE pj.user_id = $1
		  AND pj.type = 'wake_receipt'
		  AND pj.status = 'pending'
		  AND wp.status = 'scheduled'
		  AND wp.final_deadline_at >= $2
		ORDER BY wp.wake_at ASC, pj.id ASC
		LIMIT $3
	`, input.UserID, input.Now.UTC(), input.Limit)
	if err != nil {
		return nil, fmt.Errorf("receiptworker get pending wake receipt tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]WakeReceiptTask, 0, input.Limit)

	for rows.Next() {
		var task WakeReceiptTask
		if err := rows.Scan(append(printJobScanDest(&task.PrintJob), wakePlanScanDest(&task.WakePlan)...)...); err != nil {
			return nil, fmt.Errorf("receiptworker scan wake receipt task: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("receiptworker wake receipt rows: %w", err)
	}

	return tasks, nil
}

func (s *Postgres) GetPendingFinalReportTasks(ctx context.Context, input GetPendingTasksInput) ([]FinalReportTask, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("receiptworker postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return nil, errors.New("receiptworker get pending final report tasks: user_id must be > 0")
	}
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	if input.Limit <= 0 {
		input.Limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			pj.id,
			pj.user_id,
			pj.wake_plan_id,
			pj.type,
			pj.status,
			pj.not_before,
			COALESCE(pj.payload_type, 'text/plain') AS payload_type,
			COALESCE(pj.payload_text, '') AS payload_text,
			pj.error_message,
			pj.created_at,
			pj.updated_at,

			wp.id,
			wp.user_id,
			wp.date,
			wp.wake_at,
			wp.prepare_at,
			wp.final_deadline_at,
			wp.status,
			wp.source,
			wp.wake_receipt_job_id,
			wp.final_report_job_id,
			wp.fallback_job_id,
			wp.created_at,
			wp.updated_at,

			dhs.id,
			dhs.user_id,
			dhs.date,
			dhs.data_state,
			dhs.sleep_score,
			dhs.recovery_score,
			dhs.day_strain,
			dhs.sleep_minutes,
			dhs.sleep_needed_minutes,
			dhs.sleep_vs_need_pct,
			dhs.awake_minutes,
			dhs.light_sleep_minutes,
			dhs.deep_sleep_minutes,
			dhs.rem_sleep_minutes,
			dhs.restorative_sleep_minutes,
			dhs.sleep_efficiency_pct,
			dhs.sleep_consistency_pct,
			dhs.respiratory_rate,
			dhs.hrv_rmssd_ms,
			dhs.resting_heart_rate_bpm,
			dhs.spo2_pct,
			dhs.skin_temp_celsius,
			dhs.source_updated_at,
			dhs.created_at,
			dhs.updated_at,

			ma.id,
			ma.user_id,
			ma.date,
			ma.snapshot_id,
			ma.status,
			ma.model,
			ma.prompt_version,
			ma.main_signal,
			ma.day_type,
			ma.advice_json::text,
			ma.rendered_text,
			ma.motto,
			ma.error_message,
			ma.created_at,
			ma.updated_at
		FROM print_jobs pj
		JOIN wake_plans wp ON wp.id = pj.wake_plan_id

		LEFT JOIN LATERAL (
			SELECT *
			FROM daily_health_snapshots
			WHERE user_id = wp.user_id
			  AND date = wp.date
			  AND data_state = 'ready'
			ORDER BY updated_at DESC, id DESC
			LIMIT 1
		) dhs ON TRUE

		LEFT JOIN LATERAL (
			SELECT *
			FROM morning_advice
			WHERE user_id = wp.user_id
			  AND date = wp.date
			  AND status = 'ready'
			ORDER BY updated_at DESC, id DESC
			LIMIT 1
		) ma ON TRUE

		WHERE pj.user_id = $1
		  AND pj.type = 'final_report'
		  AND pj.status = 'pending'
		  AND wp.status = 'scheduled'
		  AND wp.wake_at <= $2
		ORDER BY wp.wake_at ASC, pj.id ASC
		LIMIT $3
	`, input.UserID, input.Now.UTC(), input.Limit)
	if err != nil {
		return nil, fmt.Errorf("receiptworker get pending final report tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]FinalReportTask, 0, input.Limit)

	for rows.Next() {
		task, err := scanFinalReportTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("receiptworker final report rows: %w", err)
	}

	return tasks, nil
}

func (s *Postgres) EnsureFinalReportJobs(ctx context.Context, input EnsureFinalReportJobsInput) (int, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("receiptworker postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return 0, errors.New("receiptworker ensure final report jobs: user_id must be > 0")
	}
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	if input.Limit <= 0 {
		input.Limit = 20
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("receiptworker begin ensure final report jobs tx: %w", err)
	}
	defer rollbackSilently(tx)

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			wake_at
		FROM wake_plans
		WHERE user_id = $1
		  AND status = 'scheduled'
		  AND final_report_job_id IS NULL
		  AND final_deadline_at >= $2
		ORDER BY wake_at ASC, id ASC
		LIMIT $3
		FOR UPDATE SKIP LOCKED
	`, input.UserID, input.Now.UTC(), input.Limit)
	if err != nil {
		return 0, fmt.Errorf("receiptworker select wake plans without final report jobs: %w", err)
	}
	defer rows.Close()

	type candidate struct {
		wakePlanID int64
		wakeAt     time.Time
	}

	candidates := make([]candidate, 0, input.Limit)

	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.wakePlanID, &item.wakeAt); err != nil {
			return 0, fmt.Errorf("receiptworker scan final report candidate: %w", err)
		}

		candidates = append(candidates, item)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("receiptworker final report candidate rows: %w", err)
	}

	created := 0

	for _, item := range candidates {
		var printJobID int64

		err := tx.QueryRowContext(ctx, `
			INSERT INTO print_jobs (
				user_id,
				wake_plan_id,
				type,
				status,
				not_before,
				payload_type,
				payload_text,
				error_message,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				$2,
				'final_report',
				'pending',
				$3,
				'text/plain',
				'',
				NULL,
				NOW(),
				NOW()
			)
			RETURNING id
		`, input.UserID, item.wakePlanID, item.wakeAt.UTC()).Scan(&printJobID)
		if err != nil {
			return 0, fmt.Errorf("receiptworker create final report print job: %w", err)
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE wake_plans
			SET
				final_report_job_id = $1,
				updated_at = NOW()
			WHERE id = $2
			  AND final_report_job_id IS NULL
		`, printJobID, item.wakePlanID)
		if err != nil {
			return 0, fmt.Errorf("receiptworker attach final report job to wake plan: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("receiptworker read attach final report rows affected: %w", err)
		}

		if affected == 0 {
			return 0, fmt.Errorf("receiptworker attach final report job: wake_plan_id=%d was already updated", item.wakePlanID)
		}

		created++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("receiptworker commit ensure final report jobs tx: %w", err)
	}

	return created, nil
}

func (s *Postgres) MarkPrintJobReady(ctx context.Context, input MarkPrintJobReadyInput) (PrintJob, error) {
	if s == nil || s.db == nil {
		return PrintJob{}, errors.New("receiptworker postgres storage: db is nil")
	}
	if input.PrintJobID <= 0 {
		return PrintJob{}, errors.New("receiptworker mark print job ready: print_job_id must be > 0")
	}

	payloadType := strings.TrimSpace(input.PayloadType)
	if payloadType == "" {
		payloadType = string(PayloadTypeTextPlain)
	}

	payloadText := strings.TrimSpace(input.PayloadText)
	if payloadText == "" {
		return PrintJob{}, errors.New("receiptworker mark print job ready: payload_text is required")
	}

	if input.NotBefore.IsZero() {
		input.NotBefore = time.Now().UTC()
	}

	var job PrintJob
	err := s.db.QueryRowContext(ctx, `
		UPDATE print_jobs
		SET
			status = 'ready',
			not_before = $2,
			payload_type = $3,
			payload_text = $4,
			error_message = NULL,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			id,
			user_id,
			wake_plan_id,
			type,
			status,
			not_before,
			COALESCE(payload_type, 'text/plain') AS payload_type,
			COALESCE(payload_text, '') AS payload_text,
			error_message,
			created_at,
			updated_at
	`, input.PrintJobID, input.NotBefore.UTC(), payloadType, payloadText).Scan(printJobScanDest(&job)...)
	if errors.Is(err, sql.ErrNoRows) {
		return PrintJob{}, ErrNotFound
	}
	if err != nil {
		return PrintJob{}, fmt.Errorf("receiptworker mark print job ready: %w", err)
	}

	return job, nil
}

func (s *Postgres) MarkPrintJobFailed(ctx context.Context, input MarkPrintJobFailedInput) (PrintJob, error) {
	if s == nil || s.db == nil {
		return PrintJob{}, errors.New("receiptworker postgres storage: db is nil")
	}
	if input.PrintJobID <= 0 {
		return PrintJob{}, errors.New("receiptworker mark print job failed: print_job_id must be > 0")
	}

	errorMessage := strings.TrimSpace(input.ErrorMessage)
	if errorMessage == "" {
		errorMessage = "receiptworker failed to prepare print job"
	}

	var job PrintJob
	err := s.db.QueryRowContext(ctx, `
		UPDATE print_jobs
		SET
			status = 'failed',
			error_message = $2,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			id,
			user_id,
			wake_plan_id,
			type,
			status,
			not_before,
			COALESCE(payload_type, 'text/plain') AS payload_type,
			COALESCE(payload_text, '') AS payload_text,
			error_message,
			created_at,
			updated_at
	`, input.PrintJobID, errorMessage).Scan(printJobScanDest(&job)...)
	if errors.Is(err, sql.ErrNoRows) {
		return PrintJob{}, ErrNotFound
	}
	if err != nil {
		return PrintJob{}, fmt.Errorf("receiptworker mark print job failed: %w", err)
	}

	return job, nil
}

func scanFinalReportTask(rows *sql.Rows) (FinalReportTask, error) {
	var task FinalReportTask

	var snapshot nullableSnapshot
	var advice nullableAdvice

	dest := make([]any, 0, 11+13+25+15)
	dest = append(dest, printJobScanDest(&task.PrintJob)...)
	dest = append(dest, wakePlanScanDest(&task.WakePlan)...)
	dest = append(dest, snapshot.scanDest()...)
	dest = append(dest, advice.scanDest()...)

	if err := rows.Scan(dest...); err != nil {
		return FinalReportTask{}, fmt.Errorf("receiptworker scan final report task: %w", err)
	}

	if snapshot.id.Valid {
		value := snapshot.toSnapshot()
		task.Snapshot = &value
	}

	if advice.id.Valid {
		value := advice.toAdvice()
		task.Advice = &value
	}

	return task, nil
}

func printJobScanDest(p *PrintJob) []any {
	return []any{
		&p.ID,
		&p.UserID,
		&p.WakePlanID,
		&p.Type,
		&p.Status,
		&p.NotBefore,
		&p.PayloadType,
		&p.PayloadText,
		&p.ErrorMessage,
		&p.CreatedAt,
		&p.UpdatedAt,
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

type nullableSnapshot struct {
	id     sql.NullInt64
	userID sql.NullInt64

	date      sql.NullTime
	dataState sql.NullString

	sleepScore    sql.NullInt64
	recoveryScore sql.NullInt64
	dayStrain     sql.NullFloat64

	sleepMinutes       sql.NullInt64
	sleepNeededMinutes sql.NullInt64
	sleepVsNeedPct     sql.NullInt64

	awakeMinutes       sql.NullInt64
	lightSleepMinutes  sql.NullInt64
	deepSleepMinutes   sql.NullInt64
	remSleepMinutes    sql.NullInt64
	restorativeMinutes sql.NullInt64

	sleepEfficiencyPct  sql.NullFloat64
	sleepConsistencyPct sql.NullFloat64

	respiratoryRate     sql.NullFloat64
	hrvRMSSDMS          sql.NullFloat64
	restingHeartRateBPM sql.NullInt64
	spO2Pct             sql.NullFloat64
	skinTempCelsius     sql.NullFloat64

	sourceUpdatedAt sql.NullTime

	createdAt sql.NullTime
	updatedAt sql.NullTime
}

func (s *nullableSnapshot) scanDest() []any {
	return []any{
		&s.id,
		&s.userID,
		&s.date,
		&s.dataState,
		&s.sleepScore,
		&s.recoveryScore,
		&s.dayStrain,
		&s.sleepMinutes,
		&s.sleepNeededMinutes,
		&s.sleepVsNeedPct,
		&s.awakeMinutes,
		&s.lightSleepMinutes,
		&s.deepSleepMinutes,
		&s.remSleepMinutes,
		&s.restorativeMinutes,
		&s.sleepEfficiencyPct,
		&s.sleepConsistencyPct,
		&s.respiratoryRate,
		&s.hrvRMSSDMS,
		&s.restingHeartRateBPM,
		&s.spO2Pct,
		&s.skinTempCelsius,
		&s.sourceUpdatedAt,
		&s.createdAt,
		&s.updatedAt,
	}
}

func (s *nullableSnapshot) toSnapshot() DailyHealthSnapshot {
	return DailyHealthSnapshot{
		ID:     s.id.Int64,
		UserID: s.userID.Int64,

		Date:      s.date.Time,
		DataState: SnapshotDataState(s.dataState.String),

		SleepScore:    intPtrFromNullInt64(s.sleepScore),
		RecoveryScore: intPtrFromNullInt64(s.recoveryScore),
		DayStrain:     floatPtrFromNullFloat64(s.dayStrain),

		SleepMinutes:       intPtrFromNullInt64(s.sleepMinutes),
		SleepNeededMinutes: intPtrFromNullInt64(s.sleepNeededMinutes),
		SleepVsNeedPct:     intPtrFromNullInt64(s.sleepVsNeedPct),

		AwakeMinutes:       intPtrFromNullInt64(s.awakeMinutes),
		LightSleepMinutes:  intPtrFromNullInt64(s.lightSleepMinutes),
		DeepSleepMinutes:   intPtrFromNullInt64(s.deepSleepMinutes),
		REMSleepMinutes:    intPtrFromNullInt64(s.remSleepMinutes),
		RestorativeMinutes: intPtrFromNullInt64(s.restorativeMinutes),

		SleepEfficiencyPct:  floatPtrFromNullFloat64(s.sleepEfficiencyPct),
		SleepConsistencyPct: floatPtrFromNullFloat64(s.sleepConsistencyPct),

		RespiratoryRate:     floatPtrFromNullFloat64(s.respiratoryRate),
		HRVRMSSDMS:          floatPtrFromNullFloat64(s.hrvRMSSDMS),
		RestingHeartRateBPM: intPtrFromNullInt64(s.restingHeartRateBPM),
		SpO2Pct:             floatPtrFromNullFloat64(s.spO2Pct),
		SkinTempCelsius:     floatPtrFromNullFloat64(s.skinTempCelsius),

		SourceUpdatedAt: timePtrFromNullTime(s.sourceUpdatedAt),

		CreatedAt: s.createdAt.Time,
		UpdatedAt: s.updatedAt.Time,
	}
}

type nullableAdvice struct {
	id     sql.NullInt64
	userID sql.NullInt64

	date       sql.NullTime
	snapshotID sql.NullInt64

	status sql.NullString

	model         sql.NullString
	promptVersion sql.NullString

	mainSignal sql.NullString
	dayType    sql.NullString

	adviceJSONText sql.NullString
	renderedText   sql.NullString
	motto          sql.NullString

	errorMessage sql.NullString

	createdAt sql.NullTime
	updatedAt sql.NullTime
}

func (a *nullableAdvice) scanDest() []any {
	return []any{
		&a.id,
		&a.userID,
		&a.date,
		&a.snapshotID,
		&a.status,
		&a.model,
		&a.promptVersion,
		&a.mainSignal,
		&a.dayType,
		&a.adviceJSONText,
		&a.renderedText,
		&a.motto,
		&a.errorMessage,
		&a.createdAt,
		&a.updatedAt,
	}
}

func (a *nullableAdvice) toAdvice() MorningAdvice {
	adviceJSON := json.RawMessage(`{}`)
	if strings.TrimSpace(a.adviceJSONText.String) != "" && json.Valid([]byte(a.adviceJSONText.String)) {
		adviceJSON = json.RawMessage(a.adviceJSONText.String)
	}

	return MorningAdvice{
		ID:     a.id.Int64,
		UserID: a.userID.Int64,

		Date:       a.date.Time,
		SnapshotID: a.snapshotID.Int64,

		Status: MorningAdviceStatus(a.status.String),

		Model:         a.model.String,
		PromptVersion: a.promptVersion.String,

		MainSignal: a.mainSignal.String,
		DayType:    a.dayType.String,

		AdviceJSON:   adviceJSON,
		RenderedText: a.renderedText.String,
		Motto:        a.motto.String,

		ErrorMessage: stringPtrFromNullString(a.errorMessage),

		CreatedAt: a.createdAt.Time,
		UpdatedAt: a.updatedAt.Time,
	}
}

func intPtrFromNullInt64(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}

	v := int(value.Int64)
	return &v
}

func floatPtrFromNullFloat64(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}

	v := value.Float64
	return &v
}

func timePtrFromNullTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	v := value.Time
	return &v
}

func stringPtrFromNullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	v := value.String
	return &v
}

func rollbackSilently(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}
