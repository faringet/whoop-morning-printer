package storage

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("printergateway storage: not found")

type Store interface {
	Close() error

	ClaimReadyPrintJobs(ctx context.Context, input ClaimReadyPrintJobsInput) ([]PrintJob, error)

	MarkPrintJobPrinted(ctx context.Context, input MarkPrintJobPrintedInput) (PrintJob, error)
	MarkPrintJobFailed(ctx context.Context, input MarkPrintJobFailedInput) (PrintJob, error)

	CompleteWakePlanIfPrinted(ctx context.Context, input CompleteWakePlanIfPrintedInput) (WakePlanCompletionResult, error)

	GetNextWakePlan(ctx context.Context, input GetNextWakePlanInput) (*WakePlan, error)
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

type GetNextWakePlanInput struct {
	UserID int64

	Now time.Time

	Lookahead time.Duration
}

type WakePlanCompletionResult struct {
	WakePlanID int64  `json:"wake_plan_id"`
	Completed  bool   `json:"completed"`
	Status     string `json:"status"`
}

type WakePlan struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`

	WakeAt time.Time `json:"wake_at"`

	Status string `json:"status"`

	WakeReceiptJobID *int64 `json:"wake_receipt_job_id,omitempty"`
	FinalReportJobID *int64 `json:"final_report_job_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (p WakePlan) IsScheduled() bool {
	return strings.TrimSpace(p.Status) == "scheduled"
}

type PrintJob struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`

	WakePlanID *int64 `json:"wake_plan_id,omitempty"`

	Type   PrintJobType   `json:"type"`
	Status PrintJobStatus `json:"status"`

	NotBefore time.Time `json:"not_before"`

	PayloadType string `json:"payload_type"`
	PayloadText string `json:"payload_text"`

	ClaimedBy       *string    `json:"claimed_by,omitempty"`
	ProcessingUntil *time.Time `json:"processing_until,omitempty"`

	PrintedAt *time.Time `json:"printed_at,omitempty"`
	FailedAt  *time.Time `json:"failed_at,omitempty"`

	ErrorMessage *string `json:"error_message,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (j PrintJob) IsClaimedBy(workerID string) bool {
	if j.ClaimedBy == nil {
		return false
	}

	return strings.TrimSpace(*j.ClaimedBy) == strings.TrimSpace(workerID)
}

func (j PrintJob) HasPayload() bool {
	return strings.TrimSpace(j.PayloadText) != ""
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
