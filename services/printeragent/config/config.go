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

	PrinterAgent PrinterAgent `mapstructure:"printeragent"`
	Output       Output       `mapstructure:"output"`
}

type PrinterAgent struct {
	Mode string `mapstructure:"mode"`

	UserID int64 `mapstructure:"user_id"`

	Interval time.Duration `mapstructure:"interval"`

	PollLimit int `mapstructure:"poll_limit"`

	WorkerID string `mapstructure:"worker_id"`

	ClaimTTL time.Duration `mapstructure:"claim_ttl"`

	PrintDelay time.Duration `mapstructure:"print_delay"`
}

func (p *PrinterAgent) setDefaults() {
	p.Mode = strings.ToLower(strings.TrimSpace(p.Mode))
	if p.Mode == "" {
		p.Mode = "once"
	}

	if p.Interval <= 0 {
		p.Interval = 5 * time.Second
	}

	if p.PollLimit <= 0 {
		p.PollLimit = 5
	}

	p.WorkerID = strings.TrimSpace(p.WorkerID)

	if p.ClaimTTL <= 0 {
		p.ClaimTTL = 2 * time.Minute
	}

	if p.PrintDelay < 0 {
		p.PrintDelay = 0
	}
}

func (p *PrinterAgent) Validate() error {
	if p == nil {
		return errors.New("printeragent config is nil")
	}

	switch p.Mode {
	case "once", "interval":
	default:
		return fmt.Errorf("printeragent.mode must be one of [once, interval], got %q", p.Mode)
	}

	if p.UserID <= 0 {
		return errors.New("printeragent.user_id must be > 0")
	}

	if p.Interval <= 0 {
		return errors.New("printeragent.interval must be > 0")
	}

	if p.PollLimit <= 0 {
		return errors.New("printeragent.poll_limit must be > 0")
	}

	if p.ClaimTTL <= 0 {
		return errors.New("printeragent.claim_ttl must be > 0")
	}

	if p.PrintDelay < 0 {
		return errors.New("printeragent.print_delay must be >= 0")
	}

	return nil
}

type Output struct {
	Mode string `mapstructure:"mode"`

	Dir string `mapstructure:"dir"`

	CreateDirs *bool `mapstructure:"create_dirs"`

	TrailingBlankLines int `mapstructure:"trailing_blank_lines"`

	PrinterName string `mapstructure:"printer_name"`

	CPI int `mapstructure:"cpi"`

	LPI int `mapstructure:"lpi"`

	SpoolDir string `mapstructure:"spool_dir"`

	KeepSpoolFiles *bool `mapstructure:"keep_spool_files"`
}

func (o *Output) setDefaults() {
	o.Mode = strings.ToLower(strings.TrimSpace(o.Mode))
	if o.Mode == "" {
		o.Mode = "file"
	}

	o.Dir = strings.TrimSpace(o.Dir)
	if o.Dir == "" {
		o.Dir = "./out/receipts"
	}

	o.PrinterName = strings.TrimSpace(o.PrinterName)

	if o.CPI <= 0 {
		o.CPI = 16
	}

	if o.LPI <= 0 {
		o.LPI = 8
	}

	o.SpoolDir = strings.TrimSpace(o.SpoolDir)
	if o.SpoolDir == "" {
		o.SpoolDir = "./out/print-spool"
	}

	if o.TrailingBlankLines < 0 {
		o.TrailingBlankLines = 0
	}
}

func (o *Output) Validate() error {
	if o == nil {
		return errors.New("output config is nil")
	}

	switch o.Mode {
	case "file", "stdout", "printer":
	default:
		return fmt.Errorf("output.mode must be one of [file, stdout, printer], got %q", o.Mode)
	}

	if o.Mode == "file" && strings.TrimSpace(o.Dir) == "" {
		return errors.New("output.dir is required for file mode")
	}

	if o.Mode == "printer" {
		if strings.TrimSpace(o.PrinterName) == "" {
			return errors.New("output.printer_name is required for printer mode")
		}

		if o.CPI <= 0 {
			return errors.New("output.cpi must be > 0 for printer mode")
		}

		if o.LPI <= 0 {
			return errors.New("output.lpi must be > 0 for printer mode")
		}

		if strings.TrimSpace(o.SpoolDir) == "" {
			return errors.New("output.spool_dir is required for printer mode")
		}
	}

	if o.TrailingBlankLines < 0 {
		return errors.New("output.trailing_blank_lines must be >= 0")
	}

	return nil
}

func (o Output) ShouldCreateDirs() bool {
	return boolDefault(o.CreateDirs, true)
}

func (o Output) ShouldKeepSpoolFiles() bool {
	return boolDefault(o.KeepSpoolFiles, false)
}

func (c *Config) setDefaults() {
	if c.Base.AppName == "" {
		c.Base.AppName = "printeragent"
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

	c.PrinterAgent.setDefaults()
	c.Output.setDefaults()
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
	if err := c.PrinterAgent.Validate(); err != nil {
		return fmt.Errorf("printeragent: %w", err)
	}
	if err := c.Output.Validate(); err != nil {
		return fmt.Errorf("output: %w", err)
	}

	return nil
}

func New() *Config {
	c := pcfg.MustLoad[Config](pcfg.Options{
		Paths: []string{
			"./services/printeragent/config",
			"./config",
			"./configs",
			"/etc/whoop-morning-printer",
		},
		Names:         []string{"printeragent", "config", "config.local"},
		Type:          "yaml",
		EnvPrefix:     "PRINTERAGENT",
		OptionalFiles: true,
	})

	if err := c.Validate(); err != nil {
		panic(fmt.Errorf("invalid printeragent config: %w", err))
	}

	return c
}

func boolDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}

	return *value
}
