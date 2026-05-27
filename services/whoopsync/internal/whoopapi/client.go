package whoopapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ClientConfig struct {
	BaseURL     string
	HTTPTimeout time.Duration
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg ClientConfig) (*Client, error) {
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("whoop api: base url is required")
	}
	if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
		return nil, fmt.Errorf("whoop api: invalid base url: %w", err)
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 30 * time.Second
	}

	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
	}, nil
}

func (c *Client) GetCollection(ctx context.Context, accessToken string, endpoint string, query url.Values) ([]json.RawMessage, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, fmt.Errorf("whoop api: access token is required")
	}

	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("whoop api: endpoint is required")
	}

	baseQuery := cloneValues(query)
	nextToken := ""

	var all []json.RawMessage

	for {
		pageQuery := cloneValues(baseQuery)
		if nextToken != "" {
			pageQuery.Set("nextToken", nextToken)
		}

		var page CollectionResponse[json.RawMessage]
		if err := c.getJSON(ctx, accessToken, endpoint, pageQuery, &page); err != nil {
			return nil, err
		}

		all = append(all, page.Records...)

		nextToken = strings.TrimSpace(page.NextToken)
		if nextToken == "" {
			break
		}
	}

	return all, nil
}

func (c *Client) GetOne(ctx context.Context, accessToken string, endpoint string, query url.Values) (json.RawMessage, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, fmt.Errorf("whoop api: access token is required")
	}

	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("whoop api: endpoint is required")
	}

	var raw json.RawMessage
	if err := c.getJSON(ctx, accessToken, endpoint, query, &raw); err != nil {
		return nil, err
	}

	return raw, nil
}

func (c *Client) Decode(raw json.RawMessage, out any) error {
	if len(raw) == 0 {
		return fmt.Errorf("whoop api: raw json is empty")
	}
	if out == nil {
		return fmt.Errorf("whoop api: decode target is nil")
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("whoop api: decode json: %w", err)
	}

	return nil
}

func (c *Client) getJSON(ctx context.Context, accessToken string, endpoint string, query url.Values, out any) error {
	if c == nil {
		return fmt.Errorf("whoop api: client is nil")
	}
	if out == nil {
		return fmt.Errorf("whoop api: output target is nil")
	}

	requestURL, err := c.buildURL(endpoint, query)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("whoop api: create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("whoop api: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("whoop api: read response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("whoop api: decode response: %w", err)
	}

	return nil
}

func (c *Client) buildURL(endpoint string, query url.Values) (string, error) {
	if c == nil {
		return "", fmt.Errorf("whoop api: client is nil")
	}

	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("whoop api: endpoint is required")
	}

	rawURL := c.baseURL + "/" + strings.TrimLeft(endpoint, "/")

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("whoop api: build url: %w", err)
	}

	if len(query) > 0 {
		q := u.Query()
		for key, values := range query {
			for _, value := range values {
				q.Add(key, value)
			}
		}
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

func cloneValues(in url.Values) url.Values {
	out := make(url.Values, len(in))

	for key, values := range in {
		copied := make([]string, len(values))
		copy(copied, values)
		out[key] = copied
	}

	return out
}
