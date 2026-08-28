package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	pcfg "github.com/faringet/whoop-morning-printer/pkg/config"
)

type Config struct {
	Base    pcfg.Base    `mapstructure:",squash"`
	Logger  pcfg.Logger  `mapstructure:"logger"`
	Runtime pcfg.Runtime `mapstructure:"runtime"`
	Storage pcfg.Storage `mapstructure:"storage"`

	Ollama Ollama `mapstructure:"ollama"`
	Coach  Coach  `mapstructure:"coach"`
}

type Ollama struct {
	BaseURL string        `mapstructure:"base_url"`
	Timeout time.Duration `mapstructure:"timeout"`

	Model     string `mapstructure:"model"`
	KeepAlive string `mapstructure:"keep_alive"`
	Think     bool   `mapstructure:"think"`
}

func (o *Ollama) setDefaults() {
	o.BaseURL = strings.TrimSpace(o.BaseURL)
	if o.BaseURL == "" {
		o.BaseURL = "http://127.0.0.1:11434"
	}

	if o.Timeout <= 0 {
		o.Timeout = 180 * time.Second
	}

	o.Model = strings.TrimSpace(o.Model)
	if o.Model == "" {
		o.Model = "qwen2.5:7b"
	}

	o.KeepAlive = strings.TrimSpace(o.KeepAlive)
	if o.KeepAlive == "" {
		o.KeepAlive = "2h"
	}
}

func (o *Ollama) Validate() error {
	if o == nil {
		return errors.New("ollama config is nil")
	}

	if strings.TrimSpace(o.BaseURL) == "" {
		return errors.New("ollama.base_url is required")
	}
	if o.Timeout <= 0 {
		return errors.New("ollama.timeout must be > 0")
	}
	if strings.TrimSpace(o.Model) == "" {
		return errors.New("ollama.model is required")
	}

	return nil
}

type Coach struct {
	Mode         string        `mapstructure:"mode"`
	UserID       int64         `mapstructure:"user_id"`
	Timezone     string        `mapstructure:"timezone"`
	Interval     time.Duration `mapstructure:"interval"`
	PollInterval time.Duration `mapstructure:"poll_interval"`

	SnapshotLookbackDays      int  `mapstructure:"snapshot_lookback_days"`
	RequireReadySnapshot      bool `mapstructure:"require_ready_snapshot"`
	AllowPartialAfterDeadline bool `mapstructure:"allow_partial_after_deadline"`

	WarmupOnStart     bool          `mapstructure:"warmup_on_start"`
	WarmupBeforeWake  time.Duration `mapstructure:"warmup_before_wake"`
	WarmupTimeout     time.Duration `mapstructure:"warmup_timeout"`
	MinWarmupInterval time.Duration `mapstructure:"min_warmup_interval"`

	PromptVersion string `mapstructure:"prompt_version"`
	PromptPath    string `mapstructure:"prompt_path"`

	MaxRetries   int           `mapstructure:"max_retries"`
	RetryBackoff time.Duration `mapstructure:"retry_backoff"`

	MaxAdviceRunes int `mapstructure:"max_advice_runes"`
	MaxMottoRunes  int `mapstructure:"max_motto_runes"`
}

func (c *Coach) setDefaults() {
	c.Mode = strings.ToLower(strings.TrimSpace(c.Mode))
	if c.Mode == "" {
		c.Mode = "wake_watch"
	}

	c.Timezone = strings.TrimSpace(c.Timezone)
	if c.Timezone == "" {
		c.Timezone = "Europe/Moscow"
	}

	if c.Interval <= 0 {
		c.Interval = 10 * time.Minute
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 15 * time.Second
	}

	if c.SnapshotLookbackDays <= 0 {
		c.SnapshotLookbackDays = 3
	}

	c.RequireReadySnapshot = true

	if c.WarmupBeforeWake <= 0 {
		c.WarmupBeforeWake = 15 * time.Minute
	}
	if c.WarmupTimeout <= 0 {
		c.WarmupTimeout = 2 * time.Minute
	}
	if c.MinWarmupInterval <= 0 {
		c.MinWarmupInterval = 30 * time.Minute
	}

	c.PromptVersion = strings.TrimSpace(c.PromptVersion)
	if c.PromptVersion == "" {
		c.PromptVersion = "coach_morning_v1"
	}

	if c.MaxRetries <= 0 {
		c.MaxRetries = 2
	}
	if c.RetryBackoff <= 0 {
		c.RetryBackoff = 750 * time.Millisecond
	}

	if c.MaxAdviceRunes <= 0 {
		c.MaxAdviceRunes = 900
	}
	if c.MaxMottoRunes <= 0 {
		c.MaxMottoRunes = 180
	}
}

func (c *Coach) Validate() error {
	if c == nil {
		return errors.New("coach config is nil")
	}

	switch c.Mode {
	case "once", "interval", "wake_watch":
	default:
		return fmt.Errorf("coach.mode must be one of [once, interval, wake_watch], got %q", c.Mode)
	}

	if c.UserID <= 0 {
		return errors.New("coach.user_id must be > 0")
	}

	if strings.TrimSpace(c.Timezone) == "" {
		return errors.New("coach.timezone is required")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return fmt.Errorf("coach.timezone is invalid: %w", err)
	}

	if c.Interval <= 0 {
		return errors.New("coach.interval must be > 0")
	}
	if c.PollInterval <= 0 {
		return errors.New("coach.poll_interval must be > 0")
	}
	if c.SnapshotLookbackDays <= 0 {
		return errors.New("coach.snapshot_lookback_days must be > 0")
	}
	if c.WarmupBeforeWake <= 0 {
		return errors.New("coach.warmup_before_wake must be > 0")
	}
	if c.WarmupTimeout <= 0 {
		return errors.New("coach.warmup_timeout must be > 0")
	}
	if c.MinWarmupInterval <= 0 {
		return errors.New("coach.min_warmup_interval must be > 0")
	}
	if strings.TrimSpace(c.PromptVersion) == "" {
		return errors.New("coach.prompt_version is required")
	}
	if c.MaxRetries < 0 {
		return errors.New("coach.max_retries must be >= 0")
	}
	if c.RetryBackoff < 0 {
		return errors.New("coach.retry_backoff must be >= 0")
	}
	if c.MaxAdviceRunes <= 0 {
		return errors.New("coach.max_advice_runes must be > 0")
	}
	if c.MaxMottoRunes <= 0 {
		return errors.New("coach.max_motto_runes must be > 0")
	}

	return nil
}

func (c *Config) setDefaults() {
	if c.Base.AppName == "" {
		c.Base.AppName = "coach"
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

	c.Ollama.setDefaults()
	c.Coach.setDefaults()
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
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
	if err := c.Ollama.Validate(); err != nil {
		return fmt.Errorf("ollama: %w", err)
	}
	if err := c.Coach.Validate(); err != nil {
		return fmt.Errorf("coach: %w", err)
	}

	return nil
}

func New() *Config {
	c := pcfg.MustLoad[Config](pcfg.Options{
		Paths: []string{
			"./services/coach/config",
			"./config",
			"./configs",
			"/etc/whoop-morning-printer",
		},
		Names:         []string{"coach", "config", "config.local"},
		Type:          "yaml",
		EnvPrefix:     "COACH",
		OptionalFiles: true,
	})

	if err := c.Validate(); err != nil {
		panic(fmt.Errorf("invalid coach config: %w", err))
	}

	return c
}
