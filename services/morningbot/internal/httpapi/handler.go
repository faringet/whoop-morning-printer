package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/faringet/whoop-morning-printer/services/morningbot/internal/orchestrator"
	"github.com/faringet/whoop-morning-printer/services/morningbot/internal/storage"
	"github.com/gin-gonic/gin"
)

const telegramUserIDContextKey = "telegram_user_id"

type Handler struct {
	log          *slog.Logger
	orchestrator *orchestrator.Orchestrator
}

func NewHandler(orch *orchestrator.Orchestrator, log *slog.Logger) (*Handler, error) {
	if orch == nil {
		return nil, errors.New("morningbot httpapi: orchestrator is nil")
	}
	if log == nil {
		log = slog.Default()
	}

	return &Handler{
		log: log.With(
			slog.String("layer", "transport"),
			slog.String("module", "morningbot.httpapi"),
		),
		orchestrator: orch,
	}, nil
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{
		Service: "morningbot",
		Status:  "ok",
	})
}

func (h *Handler) GetWakePlan(c *gin.Context) {
	telegramUserID, ok := getTelegramUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "Telegram authorization is required")
		return
	}

	result, err := h.orchestrator.Status(c.Request.Context())
	if errors.Is(err, storage.ErrNotFound) {
		writeError(c, http.StatusNotFound, "wake_plan_not_found", "Active wake plan was not found")
		return
	}
	if err != nil {
		h.log.Error(
			"get wake plan failed",
			slog.Int64("telegram_user_id", telegramUserID),
			slog.Any("err", err),
		)

		writeError(c, http.StatusInternalServerError, "internal_error", "Could not load the wake plan")
		return
	}

	c.JSON(http.StatusOK, newWakePlanResponse(result.WakePlan))
}

func (h *Handler) PutWakePlan(c *gin.Context) {
	telegramUserID, ok := getTelegramUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "Telegram authorization is required")
		return
	}

	var request saveWakePlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "Field wake_at must contain a valid RFC3339 date")
		return
	}

	result, err := h.orchestrator.ScheduleWakeAt(
		c.Request.Context(),
		orchestrator.ScheduleWakeAtInput{
			WakeAt:         request.WakeAt,
			TelegramUserID: &telegramUserID,
		},
	)
	if errors.Is(err, orchestrator.ErrWakeTimeInPast) {
		writeError(c, http.StatusBadRequest, "wake_time_in_past", "Wake time must be in the future")
		return
	}
	if errors.Is(err, orchestrator.ErrInvalidWakeTime) {
		writeError(c, http.StatusBadRequest, "invalid_wake_time", "Wake time is invalid")
		return
	}
	if err != nil {
		h.log.Error(
			"save wake plan failed",
			slog.Int64("telegram_user_id", telegramUserID),
			slog.Time("wake_at", request.WakeAt),
			slog.Any("err", err),
		)

		writeError(c, http.StatusInternalServerError, "internal_error", "Could not save the wake plan")
		return
	}

	c.JSON(http.StatusOK, newWakePlanResponse(result.WakePlan))
}

func (h *Handler) DeleteWakePlan(c *gin.Context) {
	telegramUserID, ok := getTelegramUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "Telegram authorization is required")
		return
	}

	_, err := h.orchestrator.Cancel(c.Request.Context())
	if errors.Is(err, storage.ErrNotFound) {
		writeError(c, http.StatusNotFound, "wake_plan_not_found", "Active wake plan was not found")
		return
	}
	if err != nil {
		h.log.Error(
			"cancel wake plan failed",
			slog.Int64("telegram_user_id", telegramUserID),
			slog.Any("err", err),
		)

		writeError(c, http.StatusInternalServerError, "internal_error", "Could not cancel the wake plan")
		return
	}

	c.Status(http.StatusNoContent)
}

func setTelegramUserID(c *gin.Context, telegramUserID int64) {
	c.Set(telegramUserIDContextKey, telegramUserID)
}

func getTelegramUserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get(telegramUserIDContextKey)
	if !exists {
		return 0, false
	}

	telegramUserID, ok := value.(int64)
	if !ok || telegramUserID <= 0 {
		return 0, false
	}

	return telegramUserID, true
}

func writeError(c *gin.Context, status int, code string, message string) {
	c.AbortWithStatusJSON(status, errorResponse{
		Error:   code,
		Message: message,
	})
}
