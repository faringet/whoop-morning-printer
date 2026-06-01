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

	ReceiptWorker ReceiptWorker `mapstructure:"receiptworker"`
	Receipt       Receipt       `mapstructure:"receipt"`
}

type ReceiptWorker struct {
	Mode string `mapstructure:"mode"`

	UserID int64 `mapstructure:"user_id"`

	Timezone string `mapstructure:"timezone"`

	Interval time.Duration `mapstructure:"interval"`

	PollLimit int `mapstructure:"poll_limit"`

	ProcessWakeReceipt *bool `mapstructure:"process_wake_receipt"`

	ProcessFinalReport *bool `mapstructure:"process_final_report"`

	EnsureFinalReportJobs *bool `mapstructure:"ensure_final_report_jobs"`

	FinalReportRequireAdvice *bool `mapstructure:"final_report_require_advice"`

	FallbackAfterDeadline *bool `mapstructure:"fallback_after_deadline"`
}

func (w *ReceiptWorker) setDefaults() {
	w.Mode = strings.ToLower(strings.TrimSpace(w.Mode))
	if w.Mode == "" {
		w.Mode = "once"
	}

	w.Timezone = strings.TrimSpace(w.Timezone)
	if w.Timezone == "" {
		w.Timezone = "Europe/Moscow"
	}

	if w.Interval <= 0 {
		w.Interval = 15 * time.Second
	}

	if w.PollLimit <= 0 {
		w.PollLimit = 20
	}
}

func (w *ReceiptWorker) Validate() error {
	if w == nil {
		return errors.New("receiptworker config is nil")
	}

	switch w.Mode {
	case "once", "interval":
	default:
		return fmt.Errorf("receiptworker.mode must be one of [once, interval], got %q", w.Mode)
	}

	if w.UserID <= 0 {
		return errors.New("receiptworker.user_id must be > 0")
	}

	if strings.TrimSpace(w.Timezone) == "" {
		return errors.New("receiptworker.timezone is required")
	}
	if _, err := time.LoadLocation(w.Timezone); err != nil {
		return fmt.Errorf("receiptworker.timezone is invalid: %w", err)
	}

	if w.Interval <= 0 {
		return errors.New("receiptworker.interval must be > 0")
	}

	if w.PollLimit <= 0 {
		return errors.New("receiptworker.poll_limit must be > 0")
	}

	return nil
}

func (w ReceiptWorker) ShouldProcessWakeReceipt() bool {
	return boolDefault(w.ProcessWakeReceipt, true)
}

func (w ReceiptWorker) ShouldProcessFinalReport() bool {
	return boolDefault(w.ProcessFinalReport, true)
}

func (w ReceiptWorker) ShouldEnsureFinalReportJobs() bool {
	return boolDefault(w.EnsureFinalReportJobs, true)
}

func (w ReceiptWorker) ShouldRequireAdviceForFinalReport() bool {
	return boolDefault(w.FinalReportRequireAdvice, true)
}

func (w ReceiptWorker) ShouldFallbackAfterDeadline() bool {
	return boolDefault(w.FallbackAfterDeadline, true)
}

type Receipt struct {
	Width int `mapstructure:"width"`

	LineSeparator string `mapstructure:"line_separator"`

	ArtEnabled *bool `mapstructure:"art_enabled"`

	ArtMode string `mapstructure:"art_mode"`

	MaxArtLines int `mapstructure:"max_art_lines"`
}

func (r *Receipt) setDefaults() {
	if r.Width <= 0 {
		r.Width = 42
	}

	r.LineSeparator = strings.TrimSpace(r.LineSeparator)
	if r.LineSeparator == "" {
		r.LineSeparator = "-"
	}

	r.ArtMode = strings.ToLower(strings.TrimSpace(r.ArtMode))
	if r.ArtMode == "" {
		r.ArtMode = "deterministic"
	}

	if r.MaxArtLines <= 0 {
		r.MaxArtLines = 8
	}
}

func (r *Receipt) Validate() error {
	if r == nil {
		return errors.New("receipt config is nil")
	}

	if r.Width < 24 || r.Width > 80 {
		return fmt.Errorf("receipt.width must be between 24 and 80, got %d", r.Width)
	}

	if strings.TrimSpace(r.LineSeparator) == "" {
		return errors.New("receipt.line_separator is required")
	}

	switch r.ArtMode {
	case "deterministic", "random":
	default:
		return fmt.Errorf("receipt.art_mode must be one of [deterministic, random], got %q", r.ArtMode)
	}

	if r.MaxArtLines <= 0 {
		return errors.New("receipt.max_art_lines must be > 0")
	}

	return nil
}

func (r Receipt) IsArtEnabled() bool {
	return boolDefault(r.ArtEnabled, true)
}

func (c *Config) setDefaults() {
	if c.Base.AppName == "" {
		c.Base.AppName = "receiptworker"
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

	c.ReceiptWorker.setDefaults()
	c.Receipt.setDefaults()
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
	if err := c.ReceiptWorker.Validate(); err != nil {
		return fmt.Errorf("receiptworker: %w", err)
	}
	if err := c.Receipt.Validate(); err != nil {
		return fmt.Errorf("receipt: %w", err)
	}

	return nil
}

func New() *Config {
	c := pcfg.MustLoad[Config](pcfg.Options{
		Paths: []string{
			"./services/receiptworker/config",
			"./config",
			"./configs",
			"/etc/whoop-morning-printer",
		},
		Names:         []string{"receiptworker", "config", "config.local"},
		Type:          "yaml",
		EnvPrefix:     "RECEIPTWORKER",
		OptionalFiles: true,
	})

	if err := c.Validate(); err != nil {
		panic(fmt.Errorf("invalid receiptworker config: %w", err))
	}

	return c
}

func boolDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}

	return *value
}
