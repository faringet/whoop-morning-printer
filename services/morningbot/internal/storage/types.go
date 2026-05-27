package storage

import (
	"context"
	"time"
)

type Store interface {
	Close() error

	EnsureUser(ctx context.Context, input EnsureUserInput) error

	ScheduleWakePlan(ctx context.Context, input ScheduleWakePlanInput) (ScheduleWakePlanResult, error)
	GetNearestActiveWakePlan(ctx context.Context, userID int64, now time.Time) (WakePlan, error)
	CancelNearestActiveWakePlan(ctx context.Context, userID int64, now time.Time) (WakePlan, error)

	CreateTestPrintJob(ctx context.Context, input CreateTestPrintJobInput) (PrintJob, error)
}

type EnsureUserInput struct {
	UserID         int64
	TelegramUserID *int64
	Timezone       string
}

type ScheduleWakePlanInput struct {
	UserID int64

	Date            time.Time
	WakeAt          time.Time
	PrepareAt       time.Time
	FinalDeadlineAt time.Time

	Source WakePlanSource
}

type ScheduleWakePlanResult struct {
	WakePlan       WakePlan
	WakeReceiptJob PrintJob
}

type WakePlan struct {
	ID     int64
	UserID int64

	Date time.Time

	WakeAt          time.Time
	PrepareAt       time.Time
	FinalDeadlineAt time.Time

	Status WakePlanStatus
	Source WakePlanSource

	WakeReceiptJobID *int64
	FinalReportJobID *int64
	FallbackJobID    *int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

type WakePlanStatus string

const (
	WakePlanStatusScheduled          WakePlanStatus = "scheduled"
	WakePlanStatusWakeReceiptReady   WakePlanStatus = "wake_receipt_ready"
	WakePlanStatusWakeReceiptPrinted WakePlanStatus = "wake_receipt_printed"
	WakePlanStatusWaitingWHOOP       WakePlanStatus = "waiting_whoop"
	WakePlanStatusWaitingAdvice      WakePlanStatus = "waiting_advice"
	WakePlanStatusFinalReportReady   WakePlanStatus = "final_report_ready"
	WakePlanStatusFinalReportPrinted WakePlanStatus = "final_report_printed"
	WakePlanStatusFallbackPrinted    WakePlanStatus = "fallback_printed"
	WakePlanStatusDone               WakePlanStatus = "done"
	WakePlanStatusCancelled          WakePlanStatus = "cancelled"
	WakePlanStatusFailed             WakePlanStatus = "failed"
)

type WakePlanSource string

const (
	WakePlanSourceManual   WakePlanSource = "manual"
	WakePlanSourceTelegram WakePlanSource = "telegram"
	WakePlanSourceDefault  WakePlanSource = "default"
	WakePlanSourceTest     WakePlanSource = "test"
)

type PrintJob struct {
	ID         int64
	UserID     int64
	WakePlanID *int64

	Type   PrintJobType
	Status PrintJobStatus

	NotBefore time.Time

	PayloadType string
	PayloadText *string

	ClaimedBy       *string
	ProcessingUntil *time.Time

	PrintedAt    *time.Time
	FailedAt     *time.Time
	ErrorMessage *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type PrintJobType string

const (
	PrintJobTypeWakeReceipt    PrintJobType = "wake_receipt"
	PrintJobTypeFinalReport    PrintJobType = "final_report"
	PrintJobTypeFallbackReport PrintJobType = "fallback_report"
	PrintJobTypeTest           PrintJobType = "test"
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

type CreateTestPrintJobInput struct {
	UserID int64

	NotBefore time.Time

	PayloadText string
}
