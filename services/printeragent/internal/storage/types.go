package storage

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("printeragent storage: not found")

type Store interface {
	Close() error

	ClaimReadyPrintJobs(ctx context.Context, input ClaimReadyPrintJobsInput) ([]PrintJob, error)

	MarkPrintJobPrinted(ctx context.Context, input MarkPrintJobPrintedInput) (PrintJob, error)
	MarkPrintJobFailed(ctx context.Context, input MarkPrintJobFailedInput) (PrintJob, error)

	CompleteWakePlanIfPrinted(ctx context.Context, input CompleteWakePlanIfPrintedInput) (WakePlanCompletionResult, error)
}

type ClaimReadyPrintJobsInput struct {
	UserID int64

	Now time.Time

	Limit int

	WorkerID string
	ClaimTTL time.Duration
}

type MarkPrintJobPrintedInput struct {
	PrintJobID int64

	WorkerID  string
	PrintedAt time.Time
}

type MarkPrintJobFailedInput struct {
	PrintJobID int64

	WorkerID string

	ErrorMessage string
	FailedAt     time.Time
}

type CompleteWakePlanIfPrintedInput struct {
	WakePlanID int64
}

type WakePlanCompletionResult struct {
	WakePlanID int64
	Completed  bool
	Status     string
}

type PrintJob struct {
	ID     int64
	UserID int64

	WakePlanID *int64

	Type   PrintJobType
	Status PrintJobStatus

	NotBefore time.Time

	PayloadType string
	PayloadText string

	ClaimedBy       *string
	ProcessingUntil *time.Time

	PrintedAt *time.Time
	FailedAt  *time.Time

	ErrorMessage *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (j PrintJob) IsClaimedBy(workerID string) bool {
	if j.ClaimedBy == nil {
		return false
	}

	return *j.ClaimedBy == workerID
}

func (j PrintJob) HasPayload() bool {
	return j.PayloadText != ""
}

type PrintJobType string

const (
	PrintJobTypeWakeReceipt PrintJobType = "wake_receipt"
	PrintJobTypeFinalReport PrintJobType = "final_report"
	PrintJobTypeFallback    PrintJobType = "fallback"
	PrintJobTypeTest        PrintJobType = "test"
)

type PrintJobStatus string

const (
	PrintJobStatusPending    PrintJobStatus = "pending"
	PrintJobStatusProcessing PrintJobStatus = "processing"
	PrintJobStatusReady      PrintJobStatus = "ready"
	PrintJobStatusPrinted    PrintJobStatus = "printed"
	PrintJobStatusFailed     PrintJobStatus = "failed"
	PrintJobStatusCancelled  PrintJobStatus = "cancelled"
)

const PayloadTypeTextPlain = "text/plain"
