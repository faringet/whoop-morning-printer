package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	pcfg "github.com/faringet/whoop-morning-printer/pkg/config"
)

type Config struct {
	Base    pcfg.Base    `mapstructure:",squash"`
	Logger  pcfg.Logger  `mapstructure:"logger"`
	Runtime pcfg.Runtime `mapstructure:"runtime"`
	Storage pcfg.Storage `mapstructure:"storage"`

	WhoopSync WhoopSync `mapstructure:"whoopsync"`
	Whoop     Whoop     `mapstructure:"whoop"`
}

type WhoopSync struct {
	Mode string `mapstructure:"mode"`

	UserID int64 `mapstructure:"user_id"`

	AuthorizationCode string `mapstructure:"authorization_code"`

	Interval         time.Duration `mapstructure:"interval"`
	StartupTimeout   time.Duration `mapstructure:"startup_timeout"`
	PingInterval     time.Duration `mapstructure:"ping_interval"`
	LookbackDays     int           `mapstructure:"lookback_days"`
	TokenRefreshSkew time.Duration `mapstructure:"token_refresh_skew"`
}

type Whoop struct {
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"`

	AuthURL    string `mapstructure:"auth_url"`
	TokenURL   string `mapstructure:"token_url"`
	APIBaseURL string `mapstructure:"api_base_url"`

	HTTPTimeout time.Duration `mapstructure:"http_timeout"`
}

func (c *Config) setDefaults() {
	if c.Base.AppName == "" {
		c.Base.AppName = "whoopsync"
	}
	if c.Base.Env == "" {
		c.Base.Env = "dev"
	}

	if c.Logger.Level == "" {
		c.Logger.Level = "info"
	}

	if c.Runtime.ShutdownTimeout == 0 {
		c.Runtime.ShutdownTimeout = 15 * time.Second
	}

	if c.Storage.Postgres.MaxOpenConns <= 0 {
		c.Storage.Postgres.MaxOpenConns = 10
	}
	if c.Storage.Postgres.MaxIdleConns < 0 {
		c.Storage.Postgres.MaxIdleConns = 0
	}
	if c.Storage.Postgres.ConnMaxLifetime < 0 {
		c.Storage.Postgres.ConnMaxLifetime = 0
	}
	if c.Storage.Postgres.ConnMaxIdleTime < 0 {
		c.Storage.Postgres.ConnMaxIdleTime = 0
	}

	c.WhoopSync.setDefaults()
	c.Whoop.setDefaults()
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	c.setDefaults()

	if err := c.Base.Validate(); err != nil {
		return fmt.Errorf("base: %w", err)
	}
	if err := c.Logger.Validate(); err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	if err := c.Runtime.Validate(); err != nil {
		return fmt.Errorf("runtime: %w", err)
	}
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	if err := c.WhoopSync.Validate(); err != nil {
		return fmt.Errorf("whoopsync: %w", err)
	}
	if err := c.Whoop.Validate(c.WhoopSync.Mode); err != nil {
		return fmt.Errorf("whoop: %w", err)
	}

	return nil
}

func (c *WhoopSync) setDefaults() {
	c.Mode = strings.ToLower(strings.TrimSpace(c.Mode))
	if c.Mode == "" {
		c.Mode = "oauth_url"
	}

	c.AuthorizationCode = strings.TrimSpace(c.AuthorizationCode)

	if c.Interval <= 0 {
		if c.Mode == "wake_watch" {
			c.Interval = 2 * time.Minute
		} else {
			c.Interval = 30 * time.Minute
		}
	}

	if c.StartupTimeout <= 0 {
		c.StartupTimeout = 60 * time.Second
	}
	if c.PingInterval <= 0 {
		c.PingInterval = 2 * time.Second
	}
	if c.LookbackDays <= 0 {
		c.LookbackDays = 3
	}
	if c.TokenRefreshSkew <= 0 {
		c.TokenRefreshSkew = 2 * time.Minute
	}
}

func (c *WhoopSync) Validate() error {
	if c == nil {
		return fmt.Errorf("whoopsync config is nil")
	}

	switch c.Mode {
	case "oauth_url", "oauth_code", "once", "interval", "wake_watch":
	default:
		return fmt.Errorf("mode must be one of [oauth_url, oauth_code, once, interval, wake_watch], got %q", c.Mode)
	}

	if c.Mode != "oauth_url" && c.UserID <= 0 {
		return fmt.Errorf("user_id must be > 0 in %q mode", c.Mode)
	}
	if c.Mode == "oauth_code" && c.AuthorizationCode == "" {
		return fmt.Errorf("authorization_code is required in oauth_code mode")
	}
	if c.Interval <= 0 {
		return fmt.Errorf("interval must be > 0")
	}
	if c.StartupTimeout <= 0 {
		return fmt.Errorf("startup_timeout must be > 0")
	}
	if c.PingInterval <= 0 {
		return fmt.Errorf("ping_interval must be > 0")
	}
	if c.LookbackDays <= 0 {
		return fmt.Errorf("lookback_days must be > 0")
	}
	if c.TokenRefreshSkew <= 0 {
		return fmt.Errorf("token_refresh_skew must be > 0")
	}

	return nil
}

func (c *Whoop) setDefaults() {
	c.ClientID = strings.TrimSpace(c.ClientID)
	c.ClientSecret = strings.TrimSpace(c.ClientSecret)
	c.RedirectURL = strings.TrimSpace(c.RedirectURL)
	c.AuthURL = strings.TrimSpace(c.AuthURL)
	c.TokenURL = strings.TrimSpace(c.TokenURL)
	c.APIBaseURL = strings.TrimSpace(c.APIBaseURL)

	if c.AuthURL == "" {
		c.AuthURL = "https://api.prod.whoop.com/oauth/oauth2/auth"
	}
	if c.TokenURL == "" {
		c.TokenURL = "https://api.prod.whoop.com/oauth/oauth2/token"
	}
	if c.APIBaseURL == "" {
		c.APIBaseURL = "https://api.prod.whoop.com/developer/v2"
	}
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 30 * time.Second
	}
	if len(c.Scopes) == 0 {
		c.Scopes = []string{
			"offline",
			"read:sleep",
			"read:recovery",
			"read:cycles",
			"read:workout",
		}
	}

	c.Scopes = cleanScopes(c.Scopes)
}

func (c *Whoop) Validate(mode string) error {
	if c == nil {
		return fmt.Errorf("whoop config is nil")
	}

	if c.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if mode != "oauth_url" && c.ClientSecret == "" {
		return fmt.Errorf("client_secret is required in %q mode", mode)
	}
	if c.RedirectURL == "" {
		return fmt.Errorf("redirect_url is required")
	}
	if _, err := url.ParseRequestURI(c.RedirectURL); err != nil {
		return fmt.Errorf("redirect_url is invalid: %w", err)
	}
	if c.AuthURL == "" {
		return fmt.Errorf("auth_url is required")
	}
	if _, err := url.ParseRequestURI(c.AuthURL); err != nil {
		return fmt.Errorf("auth_url is invalid: %w", err)
	}
	if c.TokenURL == "" {
		return fmt.Errorf("token_url is required")
	}
	if _, err := url.ParseRequestURI(c.TokenURL); err != nil {
		return fmt.Errorf("token_url is invalid: %w", err)
	}
	if c.APIBaseURL == "" {
		return fmt.Errorf("api_base_url is required")
	}
	if _, err := url.ParseRequestURI(c.APIBaseURL); err != nil {
		return fmt.Errorf("api_base_url is invalid: %w", err)
	}
	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("http_timeout must be > 0")
	}
	if len(c.Scopes) == 0 {
		return fmt.Errorf("scopes must not be empty")
	}
	if mode != "oauth_url" && !containsScope(c.Scopes, "offline") {
		return fmt.Errorf("offline scope is required for token refresh")
	}

	return nil
}

func New() *Config {
	c := pcfg.MustLoad[Config](pcfg.Options{
		Paths: []string{
			"./services/whoopsync/config",
			"./config",
			"./configs",
			"/etc/whoop-morning-printer",
		},
		Names:         []string{"whoopsync", "config", "config.local"},
		Type:          "yaml",
		EnvPrefix:     "WHOOPSYNC",
		OptionalFiles: true,
	})

	if err := c.Validate(); err != nil {
		panic(fmt.Errorf("invalid whoopsync config: %w", err))
	}

	return c
}

func cleanScopes(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))

	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}

		seen[scope] = struct{}{}
		out = append(out, scope)
	}

	return out
}

func containsScope(scopes []string, want string) bool {
	for _, scope := range scopes {
		if scope == want {
			return true
		}
	}

	return false
}
