package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/faringet/whoop-morning-printer/services/whoopsync/internal/whoopapi"
)

type Store interface {
	Close() error

	EnsureUser(ctx context.Context, userID int64, timezone string) error

	SaveTokens(ctx context.Context, tokens Tokens) error
	GetTokens(ctx context.Context, userID int64) (Tokens, error)

	GetNearestActiveWakePlan(ctx context.Context, userID int64, now time.Time) (WakePlan, error)
	GetDailyHealthSnapshotState(ctx context.Context, userID int64, date time.Time) (DataState, error)

	UpsertRawWHOOPObject(ctx context.Context, object RawWHOOPObject) error
	UpsertDailyHealthSnapshot(ctx context.Context, snapshot DailyHealthSnapshot) error
}

type Tokens struct {
	UserID int64

	AccessToken  string
	RefreshToken string
	TokenType    string
	Scopes       []string
	ExpiresAt    time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

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

func (p WakePlan) IsInsideSyncWindow(now time.Time) bool {
	if p.ID <= 0 {
		return false
	}

	if now.IsZero() {
		now = time.Now()
	}

	now = now.UTC()

	return !now.Before(p.PrepareAt.UTC()) && !now.After(p.FinalDeadlineAt.UTC())
}

func (p WakePlan) IsBeforeSyncWindow(now time.Time) bool {
	if p.ID <= 0 {
		return false
	}

	if now.IsZero() {
		now = time.Now()
	}

	return now.UTC().Before(p.PrepareAt.UTC())
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

type RawWHOOPObject struct {
	UserID     int64
	ObjectType whoopapi.ObjectType
	WHOOPID    string

	StartAt    *time.Time
	EndAt      *time.Time
	ScoreState *whoopapi.ScoreState

	PayloadJSON json.RawMessage

	FetchedAt time.Time
}

type DataState string

const (
	DataStatePending DataState = "pending"
	DataStatePartial DataState = "partial"
	DataStateReady   DataState = "ready"
	DataStateFailed  DataState = "failed"
)

type DailyHealthSnapshot struct {
	UserID int64
	Date   time.Time

	DataState DataState

	SleepWHOOPID    *string
	RecoveryWHOOPID *string
	CycleWHOOPID    *string

	SleepScore    *int
	RecoveryScore *int
	DayStrain     *float64

	SleepMinutes        *int
	SleepNeededMinutes  *int
	SleepVsNeedPct      *int
	AwakeMinutes        *int
	LightSleepMinutes   *int
	DeepSleepMinutes    *int
	REMSleepMinutes     *int
	RestorativeMinutes  *int
	SleepEfficiencyPct  *float64
	SleepConsistencyPct *float64
	RespiratoryRate     *float64

	HRVRMSSDMS          *float64
	RestingHeartRateBPM *int
	SpO2Pct             *float64
	SkinTempCelsius     *float64

	SourceUpdatedAt *time.Time
}
