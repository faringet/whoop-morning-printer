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
	"strings"
	"time"
)

const (
	defaultHTTPTimeout = 10 * time.Second
	defaultAgentName   = "wakeplanner-legacy"
)

type HTTPConfig struct {
	BaseURL string
	Token   string

	Timeout time.Duration

	AgentName string
}

type HTTPStore struct {
	baseURL string
	token   string

	client *http.Client

	agentName string
}

func NewHTTP(cfg HTTPConfig) (*HTTPStore, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("wakeplanner http storage: base_url is required")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("wakeplanner http storage: base_url is invalid: %w", err)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return nil, fmt.Errorf("wakeplanner http storage: unsupported base_url scheme %q", parsed.Scheme)
	}

	if strings.TrimSpace(parsed.Host) == "" {
		return nil, errors.New("wakeplanner http storage: base_url host is required")
	}

	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, errors.New("wakeplanner http storage: token is required")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}

	agentName := strings.TrimSpace(cfg.AgentName)
	if agentName == "" {
		agentName = defaultAgentName
	}

	return &HTTPStore{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: timeout,
		},
		agentName: agentName,
	}, nil
}

func (s *HTTPStore) Close() error {
	return nil
}

func (s *HTTPStore) GetNextWakePlan(ctx context.Context, input GetNextWakePlanInput) (*WakePlan, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("wakeplanner http storage: client is nil")
	}
	if input.UserID <= 0 {
		return nil, errors.New("wakeplanner http get next wake plan: user_id must be > 0")
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	reqBody := getNextWakePlanRequest{
		UserID: input.UserID,
		Now:    now.UTC(),
	}

	if input.Lookahead > 0 {
		reqBody.Lookahead = input.Lookahead.String()
	}

	var respBody getNextWakePlanResponse

	if err := s.doJSON(ctx, http.MethodPost, "/v1/wake-schedule/next", reqBody, &respBody); err != nil {
		return nil, fmt.Errorf("wakeplanner http get next wake plan: %w", err)
	}

	return respBody.WakePlan, nil
}

func (s *HTTPStore) doJSON(ctx context.Context, method string, path string, body interface{}, out interface{}) error {
	if ctx == nil {
		ctx = context.Background()
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	endpoint := s.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request %s %s: %w", method, path, err)
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-WMP-Agent", s.agentName)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response %s %s: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(responseBody))
		if message == "" {
			message = resp.Status
		}

		return fmt.Errorf("unexpected status %s: %s", resp.Status, message)
	}

	if out == nil {
		return nil
	}

	if len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("decode response %s %s: %w", method, path, err)
	}

	return nil
}

type getNextWakePlanRequest struct {
	UserID int64 `json:"user_id"`

	Now       time.Time `json:"now"`
	Lookahead string    `json:"lookahead,omitempty"`
}

type getNextWakePlanResponse struct {
	WakePlan *WakePlan `json:"wake_plan"`
}
