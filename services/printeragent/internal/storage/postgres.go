package storage

import (
	"context"
	"database/sql"
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
		return nil, errors.New("printeragent postgres storage: db is nil")
	}

	return &Postgres{db: db}, nil
}

func (s *Postgres) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Postgres) ClaimReadyPrintJobs(ctx context.Context, input ClaimReadyPrintJobsInput) ([]PrintJob, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("printeragent postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return nil, errors.New("printeragent claim ready print jobs: user_id must be > 0")
	}

	workerID := strings.TrimSpace(input.WorkerID)
	if workerID == "" {
		return nil, errors.New("printeragent claim ready print jobs: worker_id is required")
	}

	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	if input.Limit <= 0 {
		input.Limit = 5
	}
	if input.ClaimTTL <= 0 {
		input.ClaimTTL = 2 * time.Minute
	}

	now := input.Now.UTC()
	processingUntil := now.Add(input.ClaimTTL)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("printeragent begin claim tx: %w", err)
	}
	defer rollbackSilently(tx)

	rows, err := tx.QueryContext(ctx, `
		WITH candidates AS (
			SELECT id
			FROM print_jobs
			WHERE user_id = $1
			  AND COALESCE(payload_text, '') <> ''
			  AND (
			  	(
			  		status = 'ready'
			  		AND not_before <= $2
			  	)
			  	OR (
			  		status = 'processing'
			  		AND processing_until IS NOT NULL
			  		AND processing_until <= $2
			  	)
			  )
			ORDER BY not_before ASC, id ASC
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		UPDATE print_jobs pj
		SET
			status = 'processing',
			claimed_by = $4,
			processing_until = $5,
			error_message = NULL,
			updated_at = NOW()
		WHERE pj.id IN (SELECT id FROM candidates)
		RETURNING
			pj.id,
			pj.user_id,
			pj.wake_plan_id,
			pj.type,
			pj.status,
			pj.not_before,
			COALESCE(pj.payload_type, 'text/plain') AS payload_type,
			COALESCE(pj.payload_text, '') AS payload_text,
			pj.claimed_by,
			pj.processing_until,
			pj.printed_at,
			pj.failed_at,
			pj.error_message,
			pj.created_at,
			pj.updated_at
	`, input.UserID, now, input.Limit, workerID, processingUntil)
	if err != nil {
		return nil, fmt.Errorf("printeragent claim ready print jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]PrintJob, 0, input.Limit)

	for rows.Next() {
		job, err := scanPrintJob(rows)
		if err != nil {
			return nil, fmt.Errorf("printeragent scan claimed print job: %w", err)
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("printeragent claimed print job rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("printeragent commit claim tx: %w", err)
	}

	return jobs, nil
}

func (s *Postgres) MarkPrintJobPrinted(ctx context.Context, input MarkPrintJobPrintedInput) (PrintJob, error) {
	if s == nil || s.db == nil {
		return PrintJob{}, errors.New("printeragent postgres storage: db is nil")
	}
	if input.PrintJobID <= 0 {
		return PrintJob{}, errors.New("printeragent mark print job printed: print_job_id must be > 0")
	}

	workerID := strings.TrimSpace(input.WorkerID)
	if workerID == "" {
		return PrintJob{}, errors.New("printeragent mark print job printed: worker_id is required")
	}

	if input.PrintedAt.IsZero() {
		input.PrintedAt = time.Now().UTC()
	}

	job, err := queryPrintJobRow(ctx, s.db, `
		UPDATE print_jobs
		SET
			status = 'printed',
			printed_at = $3,
			processing_until = NULL,
			error_message = NULL,
			updated_at = NOW()
		WHERE id = $1
		  AND status = 'processing'
		  AND claimed_by = $2
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
	`, input.PrintJobID, workerID, input.PrintedAt.UTC())
	if errors.Is(err, sql.ErrNoRows) {
		return PrintJob{}, ErrNotFound
	}
	if err != nil {
		return PrintJob{}, fmt.Errorf("printeragent mark print job printed: %w", err)
	}

	return job, nil
}

func (s *Postgres) MarkPrintJobFailed(ctx context.Context, input MarkPrintJobFailedInput) (PrintJob, error) {
	if s == nil || s.db == nil {
		return PrintJob{}, errors.New("printeragent postgres storage: db is nil")
	}
	if input.PrintJobID <= 0 {
		return PrintJob{}, errors.New("printeragent mark print job failed: print_job_id must be > 0")
	}

	workerID := strings.TrimSpace(input.WorkerID)
	if workerID == "" {
		return PrintJob{}, errors.New("printeragent mark print job failed: worker_id is required")
	}

	errorMessage := strings.TrimSpace(input.ErrorMessage)
	if errorMessage == "" {
		errorMessage = "printeragent failed to print job"
	}

	if input.FailedAt.IsZero() {
		input.FailedAt = time.Now().UTC()
	}

	job, err := queryPrintJobRow(ctx, s.db, `
		UPDATE print_jobs
		SET
			status = 'failed',
			failed_at = $3,
			processing_until = NULL,
			error_message = $4,
			updated_at = NOW()
		WHERE id = $1
		  AND status = 'processing'
		  AND claimed_by = $2
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
	`, input.PrintJobID, workerID, input.FailedAt.UTC(), errorMessage)
	if errors.Is(err, sql.ErrNoRows) {
		return PrintJob{}, ErrNotFound
	}
	if err != nil {
		return PrintJob{}, fmt.Errorf("printeragent mark print job failed: %w", err)
	}

	return job, nil
}

func (s *Postgres) CompleteWakePlanIfPrinted(ctx context.Context, input CompleteWakePlanIfPrintedInput) (WakePlanCompletionResult, error) {
	if s == nil || s.db == nil {
		return WakePlanCompletionResult{}, errors.New("printeragent postgres storage: db is nil")
	}
	if input.WakePlanID <= 0 {
		return WakePlanCompletionResult{}, errors.New("printeragent complete wake plan: wake_plan_id must be > 0")
	}

	var result WakePlanCompletionResult

	err := s.db.QueryRowContext(ctx, `
		WITH completed AS (
			SELECT
				wp.id
			FROM wake_plans wp
			JOIN print_jobs wake_job
				ON wake_job.id = wp.wake_receipt_job_id
			JOIN print_jobs final_job
				ON final_job.id = wp.final_report_job_id
			WHERE wp.id = $1
			  AND wp.status = 'scheduled'
			  AND wake_job.status = 'printed'
			  AND final_job.status = 'printed'
		)
		UPDATE wake_plans wp
		SET
			status = 'done',
			updated_at = NOW()
		WHERE wp.id IN (SELECT id FROM completed)
		RETURNING
			wp.id,
			wp.status
	`, input.WakePlanID).Scan(&result.WakePlanID, &result.Status)
	if err == nil {
		result.Completed = true
		return result, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return WakePlanCompletionResult{}, fmt.Errorf("printeragent complete wake plan: %w", err)
	}

	err = s.db.QueryRowContext(ctx, `
		SELECT
			id,
			status
		FROM wake_plans
		WHERE id = $1
	`, input.WakePlanID).Scan(&result.WakePlanID, &result.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return WakePlanCompletionResult{}, ErrNotFound
	}
	if err != nil {
		return WakePlanCompletionResult{}, fmt.Errorf("printeragent get wake plan status after completion check: %w", err)
	}

	result.Completed = false

	return result, nil
}

type printJobScanner interface {
	Scan(dest ...any) error
}

func queryPrintJobRow(ctx context.Context, db *sql.DB, query string, args ...any) (PrintJob, error) {
	row := db.QueryRowContext(ctx, query, args...)

	return scanPrintJob(row)
}

func scanPrintJob(scanner printJobScanner) (PrintJob, error) {
	var item nullablePrintJob

	if err := scanner.Scan(item.scanDest()...); err != nil {
		return PrintJob{}, err
	}

	return item.toPrintJob(), nil
}

type nullablePrintJob struct {
	id     sql.NullInt64
	userID sql.NullInt64

	wakePlanID sql.NullInt64

	jobType sql.NullString
	status  sql.NullString

	notBefore sql.NullTime

	payloadType sql.NullString
	payloadText sql.NullString

	claimedBy       sql.NullString
	processingUntil sql.NullTime

	printedAt sql.NullTime
	failedAt  sql.NullTime

	errorMessage sql.NullString

	createdAt sql.NullTime
	updatedAt sql.NullTime
}

func (j *nullablePrintJob) scanDest() []any {
	return []any{
		&j.id,
		&j.userID,
		&j.wakePlanID,
		&j.jobType,
		&j.status,
		&j.notBefore,
		&j.payloadType,
		&j.payloadText,
		&j.claimedBy,
		&j.processingUntil,
		&j.printedAt,
		&j.failedAt,
		&j.errorMessage,
		&j.createdAt,
		&j.updatedAt,
	}
}

func (j *nullablePrintJob) toPrintJob() PrintJob {
	return PrintJob{
		ID:     j.id.Int64,
		UserID: j.userID.Int64,

		WakePlanID: int64PtrFromNullInt64(j.wakePlanID),

		Type:   PrintJobType(j.jobType.String),
		Status: PrintJobStatus(j.status.String),

		NotBefore: j.notBefore.Time,

		PayloadType: j.payloadType.String,
		PayloadText: j.payloadText.String,

		ClaimedBy:       stringPtrFromNullString(j.claimedBy),
		ProcessingUntil: timePtrFromNullTime(j.processingUntil),

		PrintedAt: timePtrFromNullTime(j.printedAt),
		FailedAt:  timePtrFromNullTime(j.failedAt),

		ErrorMessage: stringPtrFromNullString(j.errorMessage),

		CreatedAt: j.createdAt.Time,
		UpdatedAt: j.updatedAt.Time,
	}
}

func int64PtrFromNullInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}

	v := value.Int64
	return &v
}

func stringPtrFromNullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	v := value.String
	return &v
}

func timePtrFromNullTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	v := value.Time
	return &v
}

func rollbackSilently(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}
