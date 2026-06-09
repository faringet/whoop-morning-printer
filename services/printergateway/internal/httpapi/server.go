package httpapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/printergateway/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type Server struct {
	log       *slog.Logger
	store     storage.Store
	authToken string
	router    *gin.Engine
}

func NewServer(log *slog.Logger, store storage.Store, authToken string) (*Server, error) {
	if log == nil {
		log = slog.Default()
	}
	if store == nil {
		return nil, errors.New("printergateway httpapi: store is nil")
	}

	authToken = strings.TrimSpace(authToken)
	if authToken == "" {
		return nil, errors.New("printergateway httpapi: auth token is required")
	}

	gin.SetMode(gin.ReleaseMode)
	binding.EnableDecoderDisallowUnknownFields = true

	s := &Server{
		log: log.With(
			slog.String("layer", "httpapi"),
			slog.String("module", "printergateway.httpapi"),
		),
		store:     store,
		authToken: authToken,
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.requestLogger())

	r.GET("/healthz", s.health)
	r.GET("/v1/health", s.health)

	v1 := r.Group("/v1", s.auth)
	v1.POST("/print-jobs/claim", s.claim)
	v1.POST("/print-jobs/:id/printed", s.markPrinted)
	v1.POST("/print-jobs/:id/failed", s.markFailed)
	v1.POST("/wake-plans/:id/complete-if-printed", s.completeWakePlan)
	v1.POST("/wake-schedule/next", s.nextWakePlan)

	s.router = r

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "printergateway",
	})
}

func (s *Server) claim(c *gin.Context) {
	var req ClaimReadyPrintJobsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	input, err := req.ToStorageInput()
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	jobs, err := s.store.ClaimReadyPrintJobs(c.Request.Context(), input)
	if err != nil {
		s.log.Error("claim print jobs failed",
			slog.Int64("user_id", input.UserID),
			slog.String("worker_id", input.WorkerID),
			slog.Any("err", err),
		)
		storageFail(c, err)
		return
	}

	if jobs == nil {
		jobs = []storage.PrintJob{}
	}

	s.log.Info("print jobs claimed",
		slog.Int64("user_id", input.UserID),
		slog.String("worker_id", input.WorkerID),
		slog.Int("count", len(jobs)),
	)

	c.JSON(http.StatusOK, ClaimReadyPrintJobsResponse{
		Jobs: jobs,
	})
}

func (s *Server) markPrinted(c *gin.Context) {
	printJobID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	var req MarkPrintJobPrintedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	input, err := req.ToStorageInput(printJobID)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	job, err := s.store.MarkPrintJobPrinted(c.Request.Context(), input)
	if err != nil {
		s.log.Error("mark print job printed failed",
			slog.Int64("print_job_id", input.PrintJobID),
			slog.String("worker_id", input.WorkerID),
			slog.Any("err", err),
		)
		storageFail(c, err)
		return
	}

	s.log.Info("print job marked printed",
		slog.Int64("print_job_id", job.ID),
		slog.Int64("user_id", job.UserID),
		slog.String("worker_id", input.WorkerID),
		slog.String("type", string(job.Type)),
	)

	c.JSON(http.StatusOK, PrintJobResponse{
		Job: job,
	})
}

func (s *Server) markFailed(c *gin.Context) {
	printJobID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	var req MarkPrintJobFailedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	input, err := req.ToStorageInput(printJobID)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	job, err := s.store.MarkPrintJobFailed(c.Request.Context(), input)
	if err != nil {
		s.log.Error("mark print job failed failed",
			slog.Int64("print_job_id", input.PrintJobID),
			slog.String("worker_id", input.WorkerID),
			slog.Any("err", err),
		)
		storageFail(c, err)
		return
	}

	s.log.Warn("print job marked failed",
		slog.Int64("print_job_id", job.ID),
		slog.Int64("user_id", job.UserID),
		slog.String("worker_id", input.WorkerID),
		slog.String("type", string(job.Type)),
		slog.String("error_message", input.ErrorMessage),
	)

	c.JSON(http.StatusOK, PrintJobResponse{
		Job: job,
	})
}

func (s *Server) completeWakePlan(c *gin.Context) {
	wakePlanID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	var req CompleteWakePlanIfPrintedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	input, err := req.ToStorageInput(wakePlanID)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	result, err := s.store.CompleteWakePlanIfPrinted(c.Request.Context(), input)
	if err != nil {
		s.log.Error("complete wake plan failed",
			slog.Int64("wake_plan_id", input.WakePlanID),
			slog.Any("err", err),
		)
		storageFail(c, err)
		return
	}

	if result.Completed {
		s.log.Info("wake plan completed",
			slog.Int64("wake_plan_id", result.WakePlanID),
			slog.String("status", result.Status),
		)
	}

	c.JSON(http.StatusOK, NewWakePlanCompletionResponse(result))
}

func (s *Server) nextWakePlan(c *gin.Context) {
	var req GetNextWakePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	input, err := req.ToStorageInput()
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}

	plan, err := s.store.GetNextWakePlan(c.Request.Context(), input)
	if err != nil {
		s.log.Error("get next wake plan failed",
			slog.Int64("user_id", input.UserID),
			slog.Time("now", input.Now),
			slog.Duration("lookahead", input.Lookahead),
			slog.Any("err", err),
		)
		storageFail(c, err)
		return
	}

	if plan == nil {
		s.log.Info("next wake plan not found",
			slog.Int64("user_id", input.UserID),
			slog.Time("now", input.Now),
			slog.Duration("lookahead", input.Lookahead),
		)

		c.JSON(http.StatusOK, GetNextWakePlanResponse{
			WakePlan: nil,
		})
		return
	}

	s.log.Info("next wake plan found",
		slog.Int64("user_id", input.UserID),
		slog.Int64("wake_plan_id", plan.ID),
		slog.Time("wake_at", plan.WakeAt),
		slog.String("status", plan.Status),
	)

	c.JSON(http.StatusOK, GetNextWakePlanResponse{
		WakePlan: plan,
	})
}

func (s *Server) auth(c *gin.Context) {
	const prefix = "Bearer "

	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		c.Abort()
		return
	}

	token := strings.TrimSpace(header[len(prefix):])
	if len(token) != len(s.authToken) ||
		subtle.ConstantTimeCompare([]byte(token), []byte(s.authToken)) != 1 {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		c.Abort()
		return
	}

	c.Next()
}

func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		if c.Request.URL.Path == "/healthz" || c.Request.URL.Path == "/v1/health" {
			return
		}

		status := c.Writer.Status()
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		s.log.LogAttrs(c.Request.Context(), level, "http request completed",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
			slog.String("agent", c.GetHeader("X-WMP-Agent")),
		)
	}
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("id must be a positive integer")
	}

	return id, nil
}

func fail(c *gin.Context, status int, err error) {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = http.StatusText(status)
	}

	c.JSON(status, ErrorResponse{
		Error: message,
	})
}

func storageFail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		fail(c, http.StatusNotFound, err)
	case errors.Is(err, context.Canceled):
		fail(c, http.StatusRequestTimeout, err)
	case errors.Is(err, context.DeadlineExceeded):
		fail(c, http.StatusGatewayTimeout, err)
	default:
		fail(c, http.StatusInternalServerError, errors.New("internal server error"))
	}
}
