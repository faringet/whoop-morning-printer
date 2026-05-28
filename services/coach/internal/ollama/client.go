package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client

	timeout     time.Duration
	keepAlive   string
	temperature float64
	topP        float64
}

type Config struct {
	BaseURL string
	Timeout time.Duration

	KeepAlive string

	Temperature float64
	TopP        float64
}

func NewClient(cfg Config) (*Client, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, errors.New("coach ollama: base_url is required")
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 180 * time.Second
	}

	if cfg.Temperature <= 0 {
		cfg.Temperature = 0.2
	}
	if cfg.TopP <= 0 {
		cfg.TopP = 0.9
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 0,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		timeout:     cfg.Timeout,
		keepAlive:   strings.TrimSpace(cfg.KeepAlive),
		temperature: cfg.Temperature,
		topP:        cfg.TopP,
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
	}, nil
}

func (c *Client) Generate(ctx context.Context, model string, prompt string) (string, error) {
	if c == nil || c.httpClient == nil {
		return "", errors.New("coach ollama: client is nil")
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return "", errors.New("coach ollama: model is required")
	}

	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", errors.New("coach ollama: prompt is required")
	}

	reqBody := generateRequest{
		Model:     model,
		Prompt:    prompt,
		Stream:    false,
		KeepAlive: c.keepAlive,
		Options: map[string]any{
			"temperature": c.temperature,
			"top_p":       c.topP,
		},
	}

	out, err := c.doGenerate(ctx, reqBody)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out.Response), nil
}

func (c *Client) Warmup(ctx context.Context, model string) error {
	if c == nil || c.httpClient == nil {
		return errors.New("coach ollama: client is nil")
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return errors.New("coach ollama: model is required")
	}

	reqBody := generateRequest{
		Model:     model,
		Prompt:    "ping",
		Stream:    false,
		KeepAlive: c.keepAlive,
		Options: map[string]any{
			"temperature": 0,
			"num_predict": 1,
		},
	}

	_, err := c.doGenerate(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("coach ollama warmup: %w", err)
	}

	return nil
}

func (c *Client) doGenerate(ctx context.Context, reqBody generateRequest) (generateResponse, error) {
	if c == nil || c.httpClient == nil {
		return generateResponse{}, errors.New("coach ollama: client is nil")
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return generateResponse{}, fmt.Errorf("coach ollama marshal request: %w", err)
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return generateResponse{}, fmt.Errorf("coach ollama create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return generateResponse{}, fmt.Errorf("coach ollama request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := readAllLimit(resp.Body, 4<<20)
	if err != nil {
		return generateResponse{}, fmt.Errorf("coach ollama read response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return generateResponse{}, parseHTTPError(resp.StatusCode, responseBody)
	}

	var out generateResponse
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return generateResponse{}, fmt.Errorf("coach ollama unmarshal response: %w", err)
	}

	if strings.TrimSpace(out.Error) != "" {
		return generateResponse{}, fmt.Errorf("coach ollama response error: %s", strings.TrimSpace(out.Error))
	}

	return out, nil
}

type generateRequest struct {
	Model     string         `json:"model"`
	Prompt    string         `json:"prompt"`
	Stream    bool           `json:"stream"`
	KeepAlive string         `json:"keep_alive,omitempty"`
	Options   map[string]any `json:"options,omitempty"`
}

type generateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	DoneReason         string `json:"done_reason,omitempty"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
	Error              string `json:"error,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func parseHTTPError(statusCode int, body []byte) error {
	var errResponse errorResponse
	if err := json.Unmarshal(body, &errResponse); err == nil {
		if msg := strings.TrimSpace(errResponse.Error); msg != "" {
			return fmt.Errorf("coach ollama http %d: %s", statusCode, msg)
		}
	}

	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = http.StatusText(statusCode)
	}

	return fmt.Errorf("coach ollama http %d: %s", statusCode, msg)
}

func readAllLimit(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, errors.New("readAllLimit: limit must be > 0")
	}

	lr := &io.LimitedReader{
		R: r,
		N: limit + 1,
	}

	body, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}

	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds limit of %d bytes", limit)
	}

	return body, nil
}
