package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type HTTPConfig struct {
	BaseURL string
	Token   string
	Timeout time.Duration

	UserID   int64
	WorkerID string
}

type HTTP struct {
	baseURL string
	token   string

	userID   int64
	workerID string

	client *http.Client
}

func NewHTTP(cfg HTTPConfig) (*HTTP, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("printeragent legacy http storage: base_url is required")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("printeragent legacy http storage: invalid base_url: %w", err)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return nil, fmt.Errorf("printeragent legacy http storage: base_url scheme must be http or https, got %q", parsed.Scheme)
	}

	if strings.TrimSpace(parsed.Host) == "" {
		return nil, errors.New("printeragent legacy http storage: base_url host is required")
	}

	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, errors.New("printeragent legacy http storage: token is required")
	}

	if cfg.UserID <= 0 {
		return nil, errors.New("printeragent legacy http storage: user_id must be > 0")
	}

	workerID := strings.TrimSpace(cfg.WorkerID)
	if workerID == "" {
		return nil, errors.New("printeragent legacy http storage: worker_id is required")
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	return &HTTP{
		baseURL: baseURL,
		token:   token,

		userID:   cfg.UserID,
		workerID: workerID,

		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

func (s *HTTP) Close() error {
	return nil
}

func (s *HTTP) ClaimReadyPrintJobs(ctx context.Context, input ClaimReadyPrintJobsInput) ([]PrintJob, error) {
	if s == nil {
		return nil, errors.New("printeragent legacy http storage: storage is nil")
	}

	if input.UserID <= 0 {
		input.UserID = s.userID
	}
	if input.UserID <= 0 {
		return nil, errors.New("printeragent legacy http claim ready print jobs: user_id must be > 0")
	}

	workerID := strings.TrimSpace(input.WorkerID)
	if workerID == "" {
		workerID = s.workerID
	}
	if workerID == "" {
		return nil, errors.New("printeragent legacy http claim ready print jobs: worker_id is required")
	}

	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	if input.Limit <= 0 {
		input.Limit = 5
	}
	if input.ClaimTTL <= 0 {
		input.ClaimTTL = 2 * time.Minute
	}

	var resp claimReadyPrintJobsResponse

	err := s.doJSON(ctx, http.MethodPost, "/v1/print-jobs/claim", claimReadyPrintJobsRequest{
		UserID:   input.UserID,
		Now:      input.Now.UTC(),
		Limit:    input.Limit,
		WorkerID: workerID,
		ClaimTTL: input.ClaimTTL.String(),
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("printeragent legacy http claim ready print jobs: %w", err)
	}

	if resp.Jobs == nil {
		return []PrintJob{}, nil
	}

	return resp.Jobs, nil
}

func (s *HTTP) MarkPrintJobPrinted(ctx context.Context, input MarkPrintJobPrintedInput) (PrintJob, error) {
	if s == nil {
		return PrintJob{}, errors.New("printeragent legacy http storage: storage is nil")
	}

	if input.PrintJobID <= 0 {
		return PrintJob{}, errors.New("printeragent legacy http mark print job printed: print_job_id must be > 0")
	}

	workerID := strings.TrimSpace(input.WorkerID)
	if workerID == "" {
		workerID = s.workerID
	}
	if workerID == "" {
		return PrintJob{}, errors.New("printeragent legacy http mark print job printed: worker_id is required")
	}

	if input.PrintedAt.IsZero() {
		input.PrintedAt = time.Now().UTC()
	}

	var resp printJobResponse

	err := s.doJSON(ctx, http.MethodPost, "/v1/print-jobs/"+strconv.FormatInt(input.PrintJobID, 10)+"/printed", markPrintJobPrintedRequest{
		WorkerID:  workerID,
		PrintedAt: input.PrintedAt.UTC(),
	}, &resp)
	if err != nil {
		return PrintJob{}, fmt.Errorf("printeragent legacy http mark print job printed: %w", err)
	}

	if resp.Job.ID <= 0 {
		return PrintJob{}, errors.New("printeragent legacy http mark print job printed: empty job in response")
	}

	return resp.Job, nil
}

func (s *HTTP) MarkPrintJobFailed(ctx context.Context, input MarkPrintJobFailedInput) (PrintJob, error) {
	if s == nil {
		return PrintJob{}, errors.New("printeragent legacy http storage: storage is nil")
	}

	if input.PrintJobID <= 0 {
		return PrintJob{}, errors.New("printeragent legacy http mark print job failed: print_job_id must be > 0")
	}

	workerID := strings.TrimSpace(input.WorkerID)
	if workerID == "" {
		workerID = s.workerID
	}
	if workerID == "" {
		return PrintJob{}, errors.New("printeragent legacy http mark print job failed: worker_id is required")
	}

	message := strings.TrimSpace(input.ErrorMessage)
	if message == "" {
		message = "printeragent legacy failed to print job"
	}

	if input.FailedAt.IsZero() {
		input.FailedAt = time.Now().UTC()
	}

	var resp printJobResponse

	err := s.doJSON(ctx, http.MethodPost, "/v1/print-jobs/"+strconv.FormatInt(input.PrintJobID, 10)+"/failed", markPrintJobFailedRequest{
		WorkerID:     workerID,
		ErrorMessage: message,
		FailedAt:     input.FailedAt.UTC(),
	}, &resp)
	if err != nil {
		return PrintJob{}, fmt.Errorf("printeragent legacy http mark print job failed: %w", err)
	}

	if resp.Job.ID <= 0 {
		return PrintJob{}, errors.New("printeragent legacy http mark print job failed: empty job in response")
	}

	return resp.Job, nil
}

func (s *HTTP) CompleteWakePlanIfPrinted(ctx context.Context, input CompleteWakePlanIfPrintedInput) (WakePlanCompletionResult, error) {
	if s == nil {
		return WakePlanCompletionResult{}, errors.New("printeragent legacy http storage: storage is nil")
	}

	if input.WakePlanID <= 0 {
		return WakePlanCompletionResult{}, errors.New("printeragent legacy http complete wake plan: wake_plan_id must be > 0")
	}

	var resp completeWakePlanIfPrintedResponse

	err := s.doJSON(ctx, http.MethodPost, "/v1/wake-plans/"+strconv.FormatInt(input.WakePlanID, 10)+"/complete-if-printed", completeWakePlanIfPrintedRequest{
		WakePlanID: input.WakePlanID,
	}, &resp)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return WakePlanCompletionResult{}, ErrNotFound
		}

		return WakePlanCompletionResult{}, fmt.Errorf("printeragent legacy http complete wake plan: %w", err)
	}

	return WakePlanCompletionResult{
		WakePlanID: resp.WakePlanID,
		Completed:  resp.Completed,
		Status:     resp.Status,
	}, nil
}

func (s *HTTP) doJSON(ctx context.Context, method string, path string, requestBody interface{}, responseBody interface{}) error {
	if s == nil {
		return errors.New("printeragent legacy http storage: storage is nil")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	var body io.Reader

	if requestBody != nil {
		data, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}

		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.url(path), body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("X-WMP-Agent", s.workerID)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, resp.Body)
		return ErrNotFound
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeHTTPError(resp)
	}

	if responseBody == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(responseBody); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}

func (s *HTTP) url(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return s.baseURL
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return s.baseURL + path
}

func decodeHTTPError(resp *http.Response) error {
	if resp == nil {
		return errors.New("printergateway returned empty response")
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("printergateway returned status %d and response body read failed: %w", resp.StatusCode, err)
	}

	message := strings.TrimSpace(string(data))

	var apiErr errorResponse
	if len(data) > 0 && json.Unmarshal(data, &apiErr) == nil {
		if strings.TrimSpace(apiErr.Error) != "" {
			message = strings.TrimSpace(apiErr.Error)
		}
		if strings.TrimSpace(apiErr.Message) != "" {
			message = strings.TrimSpace(apiErr.Message)
		}
	}

	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}

	return fmt.Errorf("printergateway returned status %d: %s", resp.StatusCode, message)
}

type claimReadyPrintJobsRequest struct {
	UserID int64 `json:"user_id"`

	Now   time.Time `json:"now"`
	Limit int       `json:"limit"`

	WorkerID string `json:"worker_id"`
	ClaimTTL string `json:"claim_ttl"`
}

type claimReadyPrintJobsResponse struct {
	Jobs []PrintJob `json:"jobs"`
}

type markPrintJobPrintedRequest struct {
	WorkerID  string    `json:"worker_id"`
	PrintedAt time.Time `json:"printed_at"`
}

type markPrintJobFailedRequest struct {
	WorkerID     string    `json:"worker_id"`
	ErrorMessage string    `json:"error_message"`
	FailedAt     time.Time `json:"failed_at"`
}

type completeWakePlanIfPrintedRequest struct {
	WakePlanID int64 `json:"wake_plan_id"`
}

type completeWakePlanIfPrintedResponse struct {
	WakePlanID int64  `json:"wake_plan_id"`
	Completed  bool   `json:"completed"`
	Status     string `json:"status"`
}

type printJobResponse struct {
	Job PrintJob `json:"job"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
