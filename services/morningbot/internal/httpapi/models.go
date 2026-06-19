package httpapi

import (
	"time"

	"github.com/faringet/whoop-morning-printer/services/morningbot/internal/storage"
)

type saveWakePlanRequest struct {
	WakeAt time.Time `json:"wake_at" binding:"required"`
}

type wakePlanResponse struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`

	WakeAt          time.Time `json:"wake_at"`
	PrepareAt       time.Time `json:"prepare_at"`
	FirstReceiptAt  time.Time `json:"first_receipt_at"`
	FinalDeadlineAt time.Time `json:"final_deadline_at"`

	Status string `json:"status"`
	Source string `json:"source"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type healthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

func newWakePlanResponse(wakePlan storage.WakePlan) wakePlanResponse {
	return wakePlanResponse{
		ID:              wakePlan.ID,
		UserID:          wakePlan.UserID,
		WakeAt:          wakePlan.WakeAt,
		PrepareAt:       wakePlan.PrepareAt,
		FirstReceiptAt:  wakePlan.WakeAt,
		FinalDeadlineAt: wakePlan.FinalDeadlineAt,
		Status:          string(wakePlan.Status),
		Source:          string(wakePlan.Source),
		CreatedAt:       wakePlan.CreatedAt,
		UpdatedAt:       wakePlan.UpdatedAt,
	}
}
