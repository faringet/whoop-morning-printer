package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrNotFound = errors.New("receiptworker storage: not found")

type Store interface {
	Close() error

	GetPendingWakeReceiptTasks(ctx context.Context, input GetPendingTasksInput) ([]WakeReceiptTask, error)
	GetPendingFinalReportTasks(ctx context.Context, input GetPendingTasksInput) ([]FinalReportTask, error)

	EnsureFinalReportJobs(ctx context.Context, input EnsureFinalReportJobsInput) (int, error)

	MarkPrintJobReady(ctx context.Context, input MarkPrintJobReadyInput) (PrintJob, error)
	MarkPrintJobFailed(ctx context.Context, input MarkPrintJobFailedInput) (PrintJob, error)
}

type GetPendingTasksInput struct {
	UserID int64
	Now    time.Time
	Limit  int
}

type EnsureFinalReportJobsInput struct {
	UserID int64
	Now    time.Time
	Limit  int
}

type MarkPrintJobReadyInput struct {
	PrintJobID int64

	PayloadType string
	PayloadText string

	NotBefore time.Time
}

type MarkPrintJobFailedInput struct {
	PrintJobID int64

	ErrorMessage string
}

type WakeReceiptTask struct {
	PrintJob PrintJob
	WakePlan WakePlan
}

type FinalReportTask struct {
	PrintJob PrintJob
	WakePlan WakePlan

	Snapshot *DailyHealthSnapshot
	Advice   *MorningAdvice
}

func (t FinalReportTask) HasReadyData(requireAdvice bool) bool {
	if t.Snapshot == nil || !t.Snapshot.IsReady() {
		return false
	}

	if requireAdvice && (t.Advice == nil || !t.Advice.IsReady()) {
		return false
	}

	return true
}

func (t FinalReportTask) IsPastDeadline(now time.Time) bool {
	if t.WakePlan.FinalDeadlineAt.IsZero() {
		return false
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	return !now.UTC().Before(t.WakePlan.FinalDeadlineAt.UTC())
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

	ErrorMessage *string

	CreatedAt time.Time
	UpdatedAt time.Time
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

type PayloadType string

const (
	PayloadTypeTextPlain PayloadType = "text/plain"
)

type WakePlan struct {
	ID     int64
	UserID int64

	Date time.Time

	WakeAt          time.Time
	PrepareAt       time.Time
	FinalDeadlineAt time.Time

	Status WakePlanStatus
	Source string

	WakeReceiptJobID *int64
	FinalReportJobID *int64
	FallbackJobID    *int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

type WakePlanStatus string

const (
	WakePlanStatusScheduled WakePlanStatus = "scheduled"
	WakePlanStatusCancelled WakePlanStatus = "cancelled"
	WakePlanStatusDone      WakePlanStatus = "done"
	WakePlanStatusFailed    WakePlanStatus = "failed"
)

type DailyHealthSnapshot struct {
	ID     int64
	UserID int64

	Date      time.Time
	DataState SnapshotDataState

	SleepScore    *int
	RecoveryScore *int
	DayStrain     *float64

	SleepMinutes       *int
	SleepNeededMinutes *int
	SleepVsNeedPct     *int

	AwakeMinutes       *int
	LightSleepMinutes  *int
	DeepSleepMinutes   *int
	REMSleepMinutes    *int
	RestorativeMinutes *int

	SleepEfficiencyPct  *float64
	SleepConsistencyPct *float64

	RespiratoryRate     *float64
	HRVRMSSDMS          *float64
	RestingHeartRateBPM *int
	SpO2Pct             *float64
	SkinTempCelsius     *float64

	SourceUpdatedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type SnapshotDataState string

const (
	SnapshotDataStatePending SnapshotDataState = "pending"
	SnapshotDataStatePartial SnapshotDataState = "partial"
	SnapshotDataStateReady   SnapshotDataState = "ready"
	SnapshotDataStateFailed  SnapshotDataState = "failed"
)

func (s DailyHealthSnapshot) IsReady() bool {
	return s.DataState == SnapshotDataStateReady
}

type MorningAdvice struct {
	ID     int64
	UserID int64

	Date       time.Time
	SnapshotID int64

	Status MorningAdviceStatus

	Model         string
	PromptVersion string

	MainSignal string
	DayType    string

	AdviceJSON   json.RawMessage
	RenderedText string
	Motto        string

	ErrorMessage *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type MorningAdviceStatus string

const (
	MorningAdviceStatusPending    MorningAdviceStatus = "pending"
	MorningAdviceStatusProcessing MorningAdviceStatus = "processing"
	MorningAdviceStatusReady      MorningAdviceStatus = "ready"
	MorningAdviceStatusFailed     MorningAdviceStatus = "failed"
)

func (a MorningAdvice) IsReady() bool {
	return a.Status == MorningAdviceStatusReady
}
