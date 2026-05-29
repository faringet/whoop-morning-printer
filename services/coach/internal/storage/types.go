package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrNotFound = errors.New("coach storage: not found")

type Store interface {
	Close() error

	GetLatestSnapshot(ctx context.Context, input GetLatestSnapshotInput) (DailyHealthSnapshot, error)
	GetSnapshotForWakePlan(ctx context.Context, input GetSnapshotForWakePlanInput) (DailyHealthSnapshot, error)

	GetActiveWakePlans(ctx context.Context, input GetActiveWakePlansInput) ([]WakePlan, error)

	GetMorningAdvice(ctx context.Context, input GetMorningAdviceInput) (MorningAdvice, error)
	UpsertMorningAdvice(ctx context.Context, input UpsertMorningAdviceInput) (MorningAdvice, error)
}

type GetLatestSnapshotInput struct {
	UserID int64

	LookbackDays int

	RequireReadySnapshot      bool
	AllowPartialAfterDeadline bool
	Deadline                  time.Time
}

type GetSnapshotForWakePlanInput struct {
	UserID int64

	WakePlanDate time.Time

	LookbackDays int

	RequireReadySnapshot      bool
	AllowPartialAfterDeadline bool
	Deadline                  time.Time
}

type GetActiveWakePlansInput struct {
	UserID int64
	Now    time.Time
	Limit  int
}

type GetMorningAdviceInput struct {
	UserID int64

	Date time.Time

	PromptVersion string
}

type UpsertMorningAdviceInput struct {
	UserID int64

	Date       time.Time
	SnapshotID int64

	Model         string
	PromptVersion string

	DayType    string
	MainSignal string
	AdviceText string
	Motto      string

	PayloadJSON json.RawMessage
	GeneratedAt time.Time
}

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

func (s DailyHealthSnapshot) IsPartial() bool {
	return s.DataState == SnapshotDataStatePartial
}

type WakePlan struct {
	ID     int64
	UserID int64

	Date time.Time

	WakeAt          time.Time
	PrepareAt       time.Time
	FinalDeadlineAt time.Time

	Status string
	Source string

	WakeReceiptJobID *int64
	FinalReportJobID *int64
	FallbackJobID    *int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (w WakePlan) WarmupAt(warmupBeforeWake time.Duration) time.Time {
	if warmupBeforeWake <= 0 {
		return w.WakeAt
	}

	return w.WakeAt.Add(-warmupBeforeWake)
}

type MorningAdvice struct {
	ID     int64
	UserID int64

	Date       time.Time
	SnapshotID int64

	Model         string
	PromptVersion string

	DayType    string
	MainSignal string
	AdviceText string
	Motto      string

	PayloadJSON json.RawMessage

	GeneratedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
