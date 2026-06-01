package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("morningbot storage: not found")

const printJobTypeFinalReport = "final_report"

type Postgres struct {
	db *sql.DB
}

func NewPostgres(db *sql.DB) (*Postgres, error) {
	if db == nil {
		return nil, errors.New("morningbot postgres storage: db is nil")
	}

	return &Postgres{db: db}, nil
}

func (s *Postgres) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Postgres) EnsureUser(ctx context.Context, input EnsureUserInput) error {
	if s == nil || s.db == nil {
		return errors.New("morningbot postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return errors.New("morningbot postgres storage: user_id must be > 0")
	}

	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "Europe/Moscow"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("morningbot postgres ensure user begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var telegramUserID any
	if input.TelegramUserID != nil && *input.TelegramUserID > 0 {
		telegramUserID = *input.TelegramUserID
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (
			id,
			telegram_user_id,
			timezone,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (id)
		DO UPDATE SET
			telegram_user_id = COALESCE(EXCLUDED.telegram_user_id, users.telegram_user_id),
			timezone = EXCLUDED.timezone,
			updated_at = NOW()
	`, input.UserID, telegramUserID, timezone); err != nil {
		return fmt.Errorf("morningbot postgres ensure user: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		SELECT setval(
			pg_get_serial_sequence('users', 'id'),
			GREATEST((SELECT COALESCE(MAX(id), 1) FROM users), 1),
			true
		)
	`); err != nil {
		return fmt.Errorf("morningbot postgres sync users id sequence: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("morningbot postgres ensure user commit tx: %w", err)
	}

	return nil
}

func (s *Postgres) ScheduleWakePlan(ctx context.Context, input ScheduleWakePlanInput) (ScheduleWakePlanResult, error) {
	if s == nil || s.db == nil {
		return ScheduleWakePlanResult{}, errors.New("morningbot postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return ScheduleWakePlanResult{}, errors.New("morningbot postgres schedule wake plan: user_id must be > 0")
	}
	if input.Date.IsZero() {
		return ScheduleWakePlanResult{}, errors.New("morningbot postgres schedule wake plan: date is required")
	}
	if input.WakeAt.IsZero() {
		return ScheduleWakePlanResult{}, errors.New("morningbot postgres schedule wake plan: wake_at is required")
	}
	if input.PrepareAt.IsZero() {
		return ScheduleWakePlanResult{}, errors.New("morningbot postgres schedule wake plan: prepare_at is required")
	}
	if input.FinalDeadlineAt.IsZero() {
		return ScheduleWakePlanResult{}, errors.New("morningbot postgres schedule wake plan: final_deadline_at is required")
	}
	if input.Source == "" {
		input.Source = WakePlanSourceTelegram
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ScheduleWakePlanResult{}, fmt.Errorf("morningbot postgres schedule wake plan begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	wakePlan, err := upsertWakePlan(ctx, tx, input)
	if err != nil {
		return ScheduleWakePlanResult{}, err
	}

	wakeReceiptJob, err := ensureWakePlanPrintJob(ctx, tx, ensurePrintJobInput{
		UserID:        input.UserID,
		WakePlanID:    wakePlan.ID,
		ExistingJobID: wakePlan.WakeReceiptJobID,
		JobType:       string(PrintJobTypeWakeReceipt),
		NotBefore:     input.WakeAt.UTC(),
	})
	if err != nil {
		return ScheduleWakePlanResult{}, fmt.Errorf("morningbot postgres ensure wake receipt print job: %w", err)
	}

	wakePlan, err = attachWakeReceiptJob(ctx, tx, wakePlan.ID, wakeReceiptJob.ID)
	if err != nil {
		return ScheduleWakePlanResult{}, err
	}

	finalReportJob, err := ensureWakePlanPrintJob(ctx, tx, ensurePrintJobInput{
		UserID:        input.UserID,
		WakePlanID:    wakePlan.ID,
		ExistingJobID: wakePlan.FinalReportJobID,
		JobType:       printJobTypeFinalReport,
		NotBefore:     input.WakeAt.UTC(),
	})
	if err != nil {
		return ScheduleWakePlanResult{}, fmt.Errorf("morningbot postgres ensure final report print job: %w", err)
	}

	wakePlan, err = attachFinalReportJob(ctx, tx, wakePlan.ID, finalReportJob.ID)
	if err != nil {
		return ScheduleWakePlanResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return ScheduleWakePlanResult{}, fmt.Errorf("morningbot postgres schedule wake plan commit tx: %w", err)
	}

	return ScheduleWakePlanResult{
		WakePlan:       wakePlan,
		WakeReceiptJob: wakeReceiptJob,
	}, nil
}

func upsertWakePlan(ctx context.Context, tx *sql.Tx, input ScheduleWakePlanInput) (WakePlan, error) {
	var wakePlan WakePlan

	err := tx.QueryRowContext(ctx, `
		INSERT INTO wake_plans (
			user_id,
			date,
			wake_at,
			prepare_at,
			final_deadline_at,
			status,
			source,
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
			$7,
			NOW(),
			NOW()
		)
		ON CONFLICT (user_id, date)
		DO UPDATE SET
			wake_at = EXCLUDED.wake_at,
			prepare_at = EXCLUDED.prepare_at,
			final_deadline_at = EXCLUDED.final_deadline_at,
			status = EXCLUDED.status,
			source = EXCLUDED.source,
			updated_at = NOW()
		RETURNING
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
	`,
		input.UserID,
		dateString(input.Date),
		input.WakeAt.UTC(),
		input.PrepareAt.UTC(),
		input.FinalDeadlineAt.UTC(),
		string(WakePlanStatusScheduled),
		string(input.Source),
	).Scan(wakePlanScanDest(&wakePlan)...)
	if err != nil {
		return WakePlan{}, fmt.Errorf("morningbot postgres upsert wake plan: %w", err)
	}

	return wakePlan, nil
}

type ensurePrintJobInput struct {
	UserID     int64
	WakePlanID int64

	ExistingJobID *int64

	JobType   string
	NotBefore time.Time
}

func ensureWakePlanPrintJob(ctx context.Context, tx *sql.Tx, input ensurePrintJobInput) (PrintJob, error) {
	jobType := strings.TrimSpace(input.JobType)
	if jobType == "" {
		return PrintJob{}, errors.New("job_type is required")
	}
	if input.UserID <= 0 {
		return PrintJob{}, errors.New("user_id must be > 0")
	}
	if input.WakePlanID <= 0 {
		return PrintJob{}, errors.New("wake_plan_id must be > 0")
	}
	if input.NotBefore.IsZero() {
		return PrintJob{}, errors.New("not_before is required")
	}

	if input.ExistingJobID != nil && *input.ExistingJobID > 0 {
		job, err := resetExistingPrintJob(ctx, tx, *input.ExistingJobID, input.NotBefore)
		if err == nil {
			return job, nil
		}

		if !errors.Is(err, sql.ErrNoRows) {
			return PrintJob{}, err
		}
	}

	var job PrintJob

	err := tx.QueryRowContext(ctx, `
		INSERT INTO print_jobs (
			user_id,
			wake_plan_id,
			type,
			status,
			not_before,
			payload_type,
			payload_text,
			claimed_by,
			processing_until,
			printed_at,
			failed_at,
			error_message,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			'text/plain',
			'',
			NULL,
			NULL,
			NULL,
			NULL,
			NULL,
			NOW(),
			NOW()
		)
		RETURNING
			id,
			user_id,
			wake_plan_id,
			type,
			status,
			not_before,
			COALESCE(payload_type, 'text/plain') AS payload_type,
			COALESCE(payload_text, '') AS payload_text,
			claimed_by,
			processing_until,
			printed_at,
			failed_at,
			error_message,
			created_at,
			updated_at
	`,
		input.UserID,
		input.WakePlanID,
		jobType,
		string(PrintJobStatusPending),
		input.NotBefore.UTC(),
	).Scan(printJobScanDest(&job)...)
	if err != nil {
		return PrintJob{}, fmt.Errorf("create %s print job: %w", jobType, err)
	}

	return job, nil
}

func resetExistingPrintJob(ctx context.Context, tx *sql.Tx, printJobID int64, notBefore time.Time) (PrintJob, error) {
	var job PrintJob

	err := tx.QueryRowContext(ctx, `
		UPDATE print_jobs
		SET
			status = $1,
			not_before = $2,
			payload_type = 'text/plain',
			payload_text = '',
			claimed_by = NULL,
			processing_until = NULL,
			printed_at = NULL,
			failed_at = NULL,
			error_message = NULL,
			updated_at = NOW()
		WHERE id = $3
		RETURNING
			id,
			user_id,
			wake_plan_id,
			type,
			status,
			not_before,
			COALESCE(payload_type, 'text/plain') AS payload_type,
			COALESCE(payload_text, '') AS payload_text,
			claimed_by,
			processing_until,
			printed_at,
			failed_at,
			error_message,
			created_at,
			updated_at
	`, string(PrintJobStatusPending), notBefore.UTC(), printJobID).Scan(printJobScanDest(&job)...)
	if err != nil {
		return PrintJob{}, err
	}

	return job, nil
}

func attachWakeReceiptJob(ctx context.Context, tx *sql.Tx, wakePlanID int64, printJobID int64) (WakePlan, error) {
	var wakePlan WakePlan

	err := tx.QueryRowContext(ctx, `
		UPDATE wake_plans
		SET
			wake_receipt_job_id = $1,
			updated_at = NOW()
		WHERE id = $2
		RETURNING
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
	`, printJobID, wakePlanID).Scan(wakePlanScanDest(&wakePlan)...)
	if err != nil {
		return WakePlan{}, fmt.Errorf("morningbot postgres attach wake receipt job: %w", err)
	}

	return wakePlan, nil
}

func attachFinalReportJob(ctx context.Context, tx *sql.Tx, wakePlanID int64, printJobID int64) (WakePlan, error) {
	var wakePlan WakePlan

	err := tx.QueryRowContext(ctx, `
		UPDATE wake_plans
		SET
			final_report_job_id = $1,
			updated_at = NOW()
		WHERE id = $2
		RETURNING
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
	`, printJobID, wakePlanID).Scan(wakePlanScanDest(&wakePlan)...)
	if err != nil {
		return WakePlan{}, fmt.Errorf("morningbot postgres attach final report job: %w", err)
	}

	return wakePlan, nil
}

func (s *Postgres) GetNearestActiveWakePlan(ctx context.Context, userID int64, now time.Time) (WakePlan, error) {
	if s == nil || s.db == nil {
		return WakePlan{}, errors.New("morningbot postgres storage: db is nil")
	}
	if userID <= 0 {
		return WakePlan{}, errors.New("morningbot postgres get wake plan: user_id must be > 0")
	}
	if now.IsZero() {
		now = time.Now()
	}

	var wakePlan WakePlan
	err := s.db.QueryRowContext(ctx, `
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
		LIMIT 1
	`, userID, now.UTC()).Scan(wakePlanScanDest(&wakePlan)...)
	if errors.Is(err, sql.ErrNoRows) {
		return WakePlan{}, ErrNotFound
	}
	if err != nil {
		return WakePlan{}, fmt.Errorf("morningbot postgres get nearest active wake plan: %w", err)
	}

	return wakePlan, nil
}

func (s *Postgres) CancelNearestActiveWakePlan(ctx context.Context, userID int64, now time.Time) (WakePlan, error) {
	if s == nil || s.db == nil {
		return WakePlan{}, errors.New("morningbot postgres storage: db is nil")
	}
	if userID <= 0 {
		return WakePlan{}, errors.New("morningbot postgres cancel wake plan: user_id must be > 0")
	}
	if now.IsZero() {
		now = time.Now()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return WakePlan{}, fmt.Errorf("morningbot postgres cancel wake plan begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var wakePlan WakePlan
	err = tx.QueryRowContext(ctx, `
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
		LIMIT 1
		FOR UPDATE
	`, userID, now.UTC()).Scan(wakePlanScanDest(&wakePlan)...)
	if errors.Is(err, sql.ErrNoRows) {
		return WakePlan{}, ErrNotFound
	}
	if err != nil {
		return WakePlan{}, fmt.Errorf("morningbot postgres find wake plan to cancel: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE print_jobs
		SET
			status = 'cancelled',
			updated_at = NOW()
		WHERE wake_plan_id = $1
		  AND status IN ('pending', 'ready')
	`, wakePlan.ID); err != nil {
		return WakePlan{}, fmt.Errorf("morningbot postgres cancel print jobs: %w", err)
	}

	err = tx.QueryRowContext(ctx, `
		UPDATE wake_plans
		SET
			status = 'cancelled',
			updated_at = NOW()
		WHERE id = $1
		RETURNING
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
	`, wakePlan.ID).Scan(wakePlanScanDest(&wakePlan)...)
	if err != nil {
		return WakePlan{}, fmt.Errorf("morningbot postgres mark wake plan cancelled: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return WakePlan{}, fmt.Errorf("morningbot postgres cancel wake plan commit tx: %w", err)
	}

	return wakePlan, nil
}

func (s *Postgres) CreateTestPrintJob(ctx context.Context, input CreateTestPrintJobInput) (PrintJob, error) {
	if s == nil || s.db == nil {
		return PrintJob{}, errors.New("morningbot postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return PrintJob{}, errors.New("morningbot postgres create test print job: user_id must be > 0")
	}

	payload := strings.TrimSpace(input.PayloadText)
	if payload == "" {
		payload = "TEST PRINT JOB\n\nЕсли ты это видишь — значит очередь печати жива. Где-то рядом Postgres довольно урчит."
	}

	if input.NotBefore.IsZero() {
		input.NotBefore = time.Now().UTC()
	}

	var job PrintJob
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO print_jobs (
			user_id,
			wake_plan_id,
			type,
			status,
			not_before,
			payload_type,
			payload_text,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			NULL,
			$2,
			$3,
			$4,
			'text/plain',
			$5,
			NOW(),
			NOW()
		)
		RETURNING
			id,
			user_id,
			wake_plan_id,
			type,
			status,
			not_before,
			COALESCE(payload_type, 'text/plain') AS payload_type,
			COALESCE(payload_text, '') AS payload_text,
			claimed_by,
			processing_until,
			printed_at,
			failed_at,
			error_message,
			created_at,
			updated_at
	`, input.UserID, string(PrintJobTypeTest), string(PrintJobStatusReady), input.NotBefore.UTC(), payload).Scan(printJobScanDest(&job)...)
	if err != nil {
		return PrintJob{}, fmt.Errorf("morningbot postgres create test print job: %w", err)
	}

	return job, nil
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

func printJobScanDest(j *PrintJob) []any {
	return []any{
		&j.ID,
		&j.UserID,
		&j.WakePlanID,
		&j.Type,
		&j.Status,
		&j.NotBefore,
		&j.PayloadType,
		&j.PayloadText,
		&j.ClaimedBy,
		&j.ProcessingUntil,
		&j.PrintedAt,
		&j.FailedAt,
		&j.ErrorMessage,
		&j.CreatedAt,
		&j.UpdatedAt,
	}
}

func dateString(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}
