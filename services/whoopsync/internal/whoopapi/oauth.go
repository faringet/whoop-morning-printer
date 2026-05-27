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

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string

	AuthURL  string
	TokenURL string

	HTTPTimeout time.Duration
}

type OAuthClient struct {
	cfg        OAuthConfig
	httpClient *http.Client
	now        func() time.Time
}

func NewOAuthClient(cfg OAuthConfig) (*OAuthClient, error) {
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)
	cfg.RedirectURL = strings.TrimSpace(cfg.RedirectURL)
	cfg.AuthURL = strings.TrimSpace(cfg.AuthURL)
	cfg.TokenURL = strings.TrimSpace(cfg.TokenURL)
	cfg.Scopes = cleanStringSlice(cfg.Scopes)

	if cfg.ClientID == "" {
		return nil, fmt.Errorf("whoop oauth: client id is required")
	}
	if cfg.RedirectURL == "" {
		return nil, fmt.Errorf("whoop oauth: redirect url is required")
	}
	if cfg.AuthURL == "" {
		return nil, fmt.Errorf("whoop oauth: auth url is required")
	}
	if cfg.TokenURL == "" {
		return nil, fmt.Errorf("whoop oauth: token url is required")
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 30 * time.Second
	}

	if _, err := url.ParseRequestURI(cfg.RedirectURL); err != nil {
		return nil, fmt.Errorf("whoop oauth: invalid redirect url: %w", err)
	}
	if _, err := url.ParseRequestURI(cfg.AuthURL); err != nil {
		return nil, fmt.Errorf("whoop oauth: invalid auth url: %w", err)
	}
	if _, err := url.ParseRequestURI(cfg.TokenURL); err != nil {
		return nil, fmt.Errorf("whoop oauth: invalid token url: %w", err)
	}

	return &OAuthClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		now: time.Now,
	}, nil
}

func (c *OAuthClient) AuthorizationURL(state string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("whoop oauth: client is nil")
	}

	u, err := url.Parse(c.cfg.AuthURL)
	if err != nil {
		return "", fmt.Errorf("whoop oauth: parse auth url: %w", err)
	}

	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", c.cfg.ClientID)
	q.Set("redirect_uri", c.cfg.RedirectURL)

	if len(c.cfg.Scopes) > 0 {
		q.Set("scope", strings.Join(c.cfg.Scopes, " "))
	}
	if strings.TrimSpace(state) != "" {
		q.Set("state", strings.TrimSpace(state))
	}

	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (TokenResponse, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return TokenResponse{}, fmt.Errorf("whoop oauth: authorization code is required")
	}
	if c.cfg.ClientSecret == "" {
		return TokenResponse{}, fmt.Errorf("whoop oauth: client secret is required")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.cfg.RedirectURL)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)

	return c.postTokenForm(ctx, form)
}

func (c *OAuthClient) RefreshToken(ctx context.Context, refreshToken string) (TokenResponse, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return TokenResponse{}, fmt.Errorf("whoop oauth: refresh token is required")
	}
	if c.cfg.ClientSecret == "" {
		return TokenResponse{}, fmt.Errorf("whoop oauth: client secret is required")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	form.Set("scope", "offline")

	return c.postTokenForm(ctx, form)
}

func (c *OAuthClient) postTokenForm(ctx context.Context, form url.Values) (TokenResponse, error) {
	if c == nil {
		return TokenResponse{}, fmt.Errorf("whoop oauth: client is nil")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.cfg.TokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("whoop oauth: create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("whoop oauth: token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("whoop oauth: read token response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return TokenResponse{}, &APIError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return TokenResponse{}, fmt.Errorf("whoop oauth: decode token response: %w", err)
	}

	token.AccessToken = strings.TrimSpace(token.AccessToken)
	token.RefreshToken = strings.TrimSpace(token.RefreshToken)
	token.TokenType = strings.TrimSpace(token.TokenType)
	token.Scope = strings.TrimSpace(token.Scope)

	if token.AccessToken == "" {
		return TokenResponse{}, fmt.Errorf("whoop oauth: token response missing access_token")
	}
	if token.RefreshToken == "" {
		return TokenResponse{}, fmt.Errorf("whoop oauth: token response missing refresh_token")
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	if token.ExpiresIn <= 0 {
		return TokenResponse{}, fmt.Errorf("whoop oauth: token response has invalid expires_in")
	}

	token.ExpiresAt = c.now().UTC().Add(time.Duration(token.ExpiresIn) * time.Second)

	return token, nil
}

func cleanStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		out = append(out, value)
	}

	return out
}
