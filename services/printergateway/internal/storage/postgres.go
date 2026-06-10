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
		return nil, errors.New("printergateway postgres storage: db is nil")
	}

	return &Postgres{
		db: db,
	}, nil
}

func (s *Postgres) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Postgres) ClaimReadyPrintJobs(ctx context.Context, input ClaimReadyPrintJobsInput) ([]PrintJob, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("printergateway postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return nil, errors.New("printergateway claim ready print jobs: user_id must be > 0")
	}

	workerID := strings.TrimSpace(input.WorkerID)
	if workerID == "" {
		return nil, errors.New("printergateway claim ready print jobs: worker_id is required")
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
		return nil, fmt.Errorf("printergateway begin claim tx: %w", err)
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
		return nil, fmt.Errorf("printergateway claim ready print jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]PrintJob, 0, input.Limit)

	for rows.Next() {
		job, err := scanPrintJob(rows)
		if err != nil {
			return nil, fmt.Errorf("printergateway scan claimed print job: %w", err)
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("printergateway claimed print job rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("printergateway commit claim tx: %w", err)
	}

	return jobs, nil
}

func (s *Postgres) MarkPrintJobPrinted(ctx context.Context, input MarkPrintJobPrintedInput) (PrintJob, error) {
	if s == nil || s.db == nil {
		return PrintJob{}, errors.New("printergateway postgres storage: db is nil")
	}
	if input.PrintJobID <= 0 {
		return PrintJob{}, errors.New("printergateway mark print job printed: print_job_id must be > 0")
	}

	workerID := strings.TrimSpace(input.WorkerID)
	if workerID == "" {
		return PrintJob{}, errors.New("printergateway mark print job printed: worker_id is required")
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
		return PrintJob{}, fmt.Errorf("printergateway mark print job printed: %w", err)
	}

	return job, nil
}

func (s *Postgres) MarkPrintJobFailed(ctx context.Context, input MarkPrintJobFailedInput) (PrintJob, error) {
	if s == nil || s.db == nil {
		return PrintJob{}, errors.New("printergateway postgres storage: db is nil")
	}
	if input.PrintJobID <= 0 {
		return PrintJob{}, errors.New("printergateway mark print job failed: print_job_id must be > 0")
	}

	workerID := strings.TrimSpace(input.WorkerID)
	if workerID == "" {
		return PrintJob{}, errors.New("printergateway mark print job failed: worker_id is required")
	}

	errorMessage := strings.TrimSpace(input.ErrorMessage)
	if errorMessage == "" {
		errorMessage = "printergateway failed to print job"
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
		return PrintJob{}, fmt.Errorf("printergateway mark print job failed: %w", err)
	}

	return job, nil
}

func (s *Postgres) CompleteWakePlanIfPrinted(ctx context.Context, input CompleteWakePlanIfPrintedInput) (WakePlanCompletionResult, error) {
	if s == nil || s.db == nil {
		return WakePlanCompletionResult{}, errors.New("printergateway postgres storage: db is nil")
	}
	if input.WakePlanID <= 0 {
		return WakePlanCompletionResult{}, errors.New("printergateway complete wake plan: wake_plan_id must be > 0")
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
		return WakePlanCompletionResult{}, fmt.Errorf("printergateway complete wake plan: %w", err)
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
		return WakePlanCompletionResult{}, fmt.Errorf("printergateway get wake plan status after completion check: %w", err)
	}

	result.Completed = false

	return result, nil
}

func (s *Postgres) GetNextWakePlan(ctx context.Context, input GetNextWakePlanInput) (*WakePlan, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("printergateway postgres storage: db is nil")
	}
	if input.UserID <= 0 {
		return nil, errors.New("printergateway get next wake plan: user_id must be > 0")
	}

	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	if input.Lookahead <= 0 {
		input.Lookahead = 36 * time.Hour
	}

	now := input.Now.UTC()
	lookaheadUntil := now.Add(input.Lookahead)

	plan, err := queryWakePlanRow(ctx, s.db, `
		SELECT
			id,
			user_id,
			wake_at,
			status,
			wake_receipt_job_id,
			final_report_job_id,
			created_at,
			updated_at
		FROM wake_plans
		WHERE user_id = $1
		  AND status = 'scheduled'
		  AND wake_at >= $2
		  AND wake_at <= $3
		ORDER BY wake_at ASC, id ASC
		LIMIT 1
	`, input.UserID, now, lookaheadUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("printergateway get next wake plan: %w", err)
	}

	return &plan, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func queryPrintJobRow(ctx context.Context, db *sql.DB, query string, args ...any) (PrintJob, error) {
	if db == nil {
		return PrintJob{}, errors.New("printergateway query print job row: db is nil")
	}

	return scanPrintJob(db.QueryRowContext(ctx, query, args...))
}

func scanPrintJob(scanner rowScanner) (PrintJob, error) {
	var job PrintJob

	if scanner == nil {
		return PrintJob{}, errors.New("printergateway scan print job: scanner is nil")
	}

	if err := scanner.Scan(
		&job.ID,
		&job.UserID,
		&job.WakePlanID,
		&job.Type,
		&job.Status,
		&job.NotBefore,
		&job.PayloadType,
		&job.PayloadText,
		&job.ClaimedBy,
		&job.ProcessingUntil,
		&job.PrintedAt,
		&job.FailedAt,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return PrintJob{}, err
	}

	return job, nil
}

func queryWakePlanRow(ctx context.Context, db *sql.DB, query string, args ...any) (WakePlan, error) {
	if db == nil {
		return WakePlan{}, errors.New("printergateway query wake plan row: db is nil")
	}

	return scanWakePlan(db.QueryRowContext(ctx, query, args...))
}

func scanWakePlan(scanner rowScanner) (WakePlan, error) {
	var plan WakePlan

	if scanner == nil {
		return WakePlan{}, errors.New("printergateway scan wake plan: scanner is nil")
	}

	if err := scanner.Scan(
		&plan.ID,
		&plan.UserID,
		&plan.WakeAt,
		&plan.Status,
		&plan.WakeReceiptJobID,
		&plan.FinalReportJobID,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	); err != nil {
		return WakePlan{}, err
	}

	return plan, nil
}

func rollbackSilently(tx *sql.Tx) {
	if tx == nil {
		return
	}

	_ = tx.Rollback()
}
