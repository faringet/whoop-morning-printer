package storage

import (
	"context"
	"strings"
	"time"
)

type Store interface {
	Close() error

	GetNextWakePlan(ctx context.Context, input GetNextWakePlanInput) (*WakePlan, error)
}

type GetNextWakePlanInput struct {
	UserID int64

	Now       time.Time
	Lookahead time.Duration
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

func (p WakePlan) HasWakeTime() bool {
	return !p.WakeAt.IsZero()
}
