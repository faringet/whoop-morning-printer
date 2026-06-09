package httpapi

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/printergateway/internal/storage"
)

type ClaimReadyPrintJobsRequest struct {
	UserID int64 `json:"user_id"`

	Now   time.Time `json:"now"`
	Limit int       `json:"limit"`

	WorkerID string `json:"worker_id"`
	ClaimTTL string `json:"claim_ttl"`
}

func (r ClaimReadyPrintJobsRequest) ToStorageInput() (storage.ClaimReadyPrintJobsInput, error) {
	if r.UserID <= 0 {
		return storage.ClaimReadyPrintJobsInput{}, errors.New("user_id must be > 0")
	}

	workerID := strings.TrimSpace(r.WorkerID)
	if workerID == "" {
		return storage.ClaimReadyPrintJobsInput{}, errors.New("worker_id is required")
	}

	now := r.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	limit := r.Limit
	if limit <= 0 {
		limit = 5
	}

	claimTTL := 2 * time.Minute

	rawClaimTTL := strings.TrimSpace(r.ClaimTTL)
	if rawClaimTTL != "" {
		parsed, err := time.ParseDuration(rawClaimTTL)
		if err != nil {
			return storage.ClaimReadyPrintJobsInput{}, fmt.Errorf("claim_ttl is invalid: %w", err)
		}

		claimTTL = parsed
	}

	if claimTTL <= 0 {
		return storage.ClaimReadyPrintJobsInput{}, errors.New("claim_ttl must be > 0")
	}

	return storage.ClaimReadyPrintJobsInput{
		UserID: r.UserID,

		Now: now.UTC(),

		Limit: limit,

		WorkerID: workerID,
		ClaimTTL: claimTTL,
	}, nil
}

type ClaimReadyPrintJobsResponse struct {
	Jobs []storage.PrintJob `json:"jobs"`
}

type MarkPrintJobPrintedRequest struct {
	WorkerID  string    `json:"worker_id"`
	PrintedAt time.Time `json:"printed_at"`
}

func (r MarkPrintJobPrintedRequest) ToStorageInput(printJobID int64) (storage.MarkPrintJobPrintedInput, error) {
	if printJobID <= 0 {
		return storage.MarkPrintJobPrintedInput{}, errors.New("print_job_id must be > 0")
	}

	workerID := strings.TrimSpace(r.WorkerID)
	if workerID == "" {
		return storage.MarkPrintJobPrintedInput{}, errors.New("worker_id is required")
	}

	printedAt := r.PrintedAt
	if printedAt.IsZero() {
		printedAt = time.Now().UTC()
	}

	return storage.MarkPrintJobPrintedInput{
		PrintJobID: printJobID,

		WorkerID:  workerID,
		PrintedAt: printedAt.UTC(),
	}, nil
}

type MarkPrintJobFailedRequest struct {
	WorkerID string `json:"worker_id"`

	ErrorMessage string    `json:"error_message"`
	FailedAt     time.Time `json:"failed_at"`
}

func (r MarkPrintJobFailedRequest) ToStorageInput(printJobID int64) (storage.MarkPrintJobFailedInput, error) {
	if printJobID <= 0 {
		return storage.MarkPrintJobFailedInput{}, errors.New("print_job_id must be > 0")
	}

	workerID := strings.TrimSpace(r.WorkerID)
	if workerID == "" {
		return storage.MarkPrintJobFailedInput{}, errors.New("worker_id is required")
	}

	errorMessage := strings.TrimSpace(r.ErrorMessage)
	if errorMessage == "" {
		errorMessage = "printeragent failed to print job"
	}

	failedAt := r.FailedAt
	if failedAt.IsZero() {
		failedAt = time.Now().UTC()
	}

	return storage.MarkPrintJobFailedInput{
		PrintJobID: printJobID,

		WorkerID: workerID,

		ErrorMessage: errorMessage,
		FailedAt:     failedAt.UTC(),
	}, nil
}

type CompleteWakePlanIfPrintedRequest struct {
	WakePlanID int64 `json:"wake_plan_id"`
}

func (r CompleteWakePlanIfPrintedRequest) ToStorageInput(wakePlanID int64) (storage.CompleteWakePlanIfPrintedInput, error) {
	if wakePlanID <= 0 {
		wakePlanID = r.WakePlanID
	}
	if wakePlanID <= 0 {
		return storage.CompleteWakePlanIfPrintedInput{}, errors.New("wake_plan_id must be > 0")
	}

	return storage.CompleteWakePlanIfPrintedInput{
		WakePlanID: wakePlanID,
	}, nil
}

type PrintJobResponse struct {
	Job storage.PrintJob `json:"job"`
}

type WakePlanCompletionResponse struct {
	WakePlanID int64  `json:"wake_plan_id"`
	Completed  bool   `json:"completed"`
	Status     string `json:"status"`
}

func NewWakePlanCompletionResponse(result storage.WakePlanCompletionResult) WakePlanCompletionResponse {
	return WakePlanCompletionResponse{
		WakePlanID: result.WakePlanID,
		Completed:  result.Completed,
		Status:     result.Status,
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}
