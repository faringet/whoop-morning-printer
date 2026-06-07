package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	raw := strings.TrimSpace(string(text))
	if raw == "" {
		d.Duration = 0
		return nil
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", raw, err)
	}

	d.Duration = parsed
	return nil
}

type Config struct {
	AppName string `yaml:"app_name"`
	Env     string `yaml:"env"`

	Logger  Logger  `yaml:"logger"`
	Runtime Runtime `yaml:"runtime"`
	Storage Storage `yaml:"storage"`

	PrinterAgent PrinterAgent `yaml:"printeragent"`
	Output       Output       `yaml:"output"`
}

type Logger struct {
	Level string `yaml:"level"`

	JSON bool `yaml:"json"`

	FileEnabled bool   `yaml:"file_enabled"`
	FilePath    string `yaml:"file_path"`
}

type Runtime struct {
	ShutdownTimeout Duration `yaml:"shutdown_timeout"`
}

type Storage struct {
	HTTP HTTPStorage `yaml:"http"`
}

type HTTPStorage struct {
	BaseURL string `yaml:"base_url"`

	Token string `yaml:"token"`

	TokenFile string `yaml:"token_file"`

	Timeout Duration `yaml:"timeout"`
}

type PrinterAgent struct {
	Mode string `yaml:"mode"`

	UserID int64 `yaml:"user_id"`

	Interval Duration `yaml:"interval"`

	PollLimit int `yaml:"poll_limit"`

	WorkerID string `yaml:"worker_id"`

	ClaimTTL Duration `yaml:"claim_ttl"`

	PrintDelay Duration `yaml:"print_delay"`
}

type Output struct {
	Mode string `yaml:"mode"`

	Dir string `yaml:"dir"`

	CreateDirs *bool `yaml:"create_dirs"`

	TrailingBlankLines int `yaml:"trailing_blank_lines"`

	PrinterName string `yaml:"printer_name"`

	CPI int `yaml:"cpi"`

	LPI int `yaml:"lpi"`

	SpoolDir string `yaml:"spool_dir"`

	KeepSpoolFiles *bool `yaml:"keep_spool_files"`
}

func Load(explicitPath string) (*Config, string, error) {
	paths := candidateConfigPaths(explicitPath)

	var tried []string

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		tried = append(tried, path)

		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, "", fmt.Errorf("read config %s: %w", path, err)
		}

		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, "", fmt.Errorf("parse config %s: %w", path, err)
		}

		cfg.setDefaults()

		if err := cfg.Validate(); err != nil {
			return nil, "", fmt.Errorf("invalid config %s: %w", path, err)
		}

		return &cfg, path, nil
	}

	return nil, "", fmt.Errorf("config file not found; tried: %s", strings.Join(tried, ", "))
}

func candidateConfigPaths(explicitPath string) []string {
	if strings.TrimSpace(explicitPath) != "" {
		return []string{explicitPath}
	}

	return []string{
		"./config/config.local.yaml",
		"./config/printeragent.yaml",
		"./printeragent.yaml",
		"./config.local.yaml",
		"/opt/whoop-morning-printer/config/config.local.yaml",
		"/etc/whoop-morning-printer/config.local.yaml",
		"/etc/whoop-morning-printer/printeragent.yaml",
	}
}

func (c *Config) setDefaults() {
	c.AppName = strings.TrimSpace(c.AppName)
	if c.AppName == "" {
		c.AppName = "printeragent-legacy"
	}

	c.Env = strings.ToLower(strings.TrimSpace(c.Env))
	if c.Env == "" {
		c.Env = "dev"
	}

	c.Logger.Level = strings.ToLower(strings.TrimSpace(c.Logger.Level))
	if c.Logger.Level == "" {
		c.Logger.Level = "info"
	}

	c.Logger.FilePath = strings.TrimSpace(c.Logger.FilePath)
	if c.Logger.FileEnabled && c.Logger.FilePath == "" {
		c.Logger.FilePath = "./logs/printeragent.log"
	}

	if c.Runtime.ShutdownTimeout.Duration <= 0 {
		c.Runtime.ShutdownTimeout.Duration = 15 * time.Second
	}

	c.Storage.HTTP.BaseURL = strings.TrimRight(strings.TrimSpace(c.Storage.HTTP.BaseURL), "/")
	c.Storage.HTTP.Token = strings.TrimSpace(c.Storage.HTTP.Token)
	c.Storage.HTTP.TokenFile = strings.TrimSpace(c.Storage.HTTP.TokenFile)

	if c.Storage.HTTP.Timeout.Duration <= 0 {
		c.Storage.HTTP.Timeout.Duration = 10 * time.Second
	}

	c.PrinterAgent.Mode = strings.ToLower(strings.TrimSpace(c.PrinterAgent.Mode))
	if c.PrinterAgent.Mode == "" {
		c.PrinterAgent.Mode = "once"
	}

	if c.PrinterAgent.Interval.Duration <= 0 {
		c.PrinterAgent.Interval.Duration = 5 * time.Second
	}

	if c.PrinterAgent.PollLimit <= 0 {
		c.PrinterAgent.PollLimit = 5
	}

	c.PrinterAgent.WorkerID = strings.TrimSpace(c.PrinterAgent.WorkerID)

	if c.PrinterAgent.ClaimTTL.Duration <= 0 {
		c.PrinterAgent.ClaimTTL.Duration = 2 * time.Minute
	}

	if c.PrinterAgent.PrintDelay.Duration < 0 {
		c.PrinterAgent.PrintDelay.Duration = 0
	}

	c.Output.Mode = strings.ToLower(strings.TrimSpace(c.Output.Mode))
	if c.Output.Mode == "" {
		c.Output.Mode = "file"
	}

	c.Output.Dir = strings.TrimSpace(c.Output.Dir)
	if c.Output.Dir == "" {
		c.Output.Dir = "./out/receipts"
	}

	c.Output.PrinterName = strings.TrimSpace(c.Output.PrinterName)

	if c.Output.CPI <= 0 {
		c.Output.CPI = 16
	}

	if c.Output.LPI <= 0 {
		c.Output.LPI = 8
	}

	c.Output.SpoolDir = strings.TrimSpace(c.Output.SpoolDir)
	if c.Output.SpoolDir == "" {
		c.Output.SpoolDir = "./out/print-spool"
	}

	if c.Output.TrailingBlankLines < 0 {
		c.Output.TrailingBlankLines = 0
	}
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}

	switch c.Logger.Level {
	case "debug", "info", "warn", "warning", "error":
	default:
		return fmt.Errorf("logger.level must be one of [debug, info, warn, warning, error], got %q", c.Logger.Level)
	}

	if err := validateHTTPStorage(c.Storage.HTTP); err != nil {
		return fmt.Errorf("storage.http: %w", err)
	}

	switch c.PrinterAgent.Mode {
	case "once", "interval":
	default:
		return fmt.Errorf("printeragent.mode must be one of [once, interval], got %q", c.PrinterAgent.Mode)
	}

	if c.PrinterAgent.UserID <= 0 {
		return errors.New("printeragent.user_id must be > 0")
	}

	if c.PrinterAgent.Interval.Duration <= 0 {
		return errors.New("printeragent.interval must be > 0")
	}

	if c.PrinterAgent.PollLimit <= 0 {
		return errors.New("printeragent.poll_limit must be > 0")
	}

	if c.PrinterAgent.ClaimTTL.Duration <= 0 {
		return errors.New("printeragent.claim_ttl must be > 0")
	}

	if c.PrinterAgent.PrintDelay.Duration < 0 {
		return errors.New("printeragent.print_delay must be >= 0")
	}

	switch c.Output.Mode {
	case "file", "stdout", "printer":
	default:
		return fmt.Errorf("output.mode must be one of [file, stdout, printer], got %q", c.Output.Mode)
	}

	if c.Output.Mode == "file" && strings.TrimSpace(c.Output.Dir) == "" {
		return errors.New("output.dir is required for file mode")
	}

	if c.Output.Mode == "printer" {
		if strings.TrimSpace(c.Output.PrinterName) == "" {
			return errors.New("output.printer_name is required for printer mode")
		}

		if c.Output.CPI <= 0 {
			return errors.New("output.cpi must be > 0 for printer mode")
		}

		if c.Output.LPI <= 0 {
			return errors.New("output.lpi must be > 0 for printer mode")
		}

		if strings.TrimSpace(c.Output.SpoolDir) == "" {
			return errors.New("output.spool_dir is required for printer mode")
		}
	}

	if c.Output.TrailingBlankLines < 0 {
		return errors.New("output.trailing_blank_lines must be >= 0")
	}

	return nil
}

func validateHTTPStorage(cfg HTTPStorage) error {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return errors.New("base_url is required")
	}

	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("base_url is invalid: %w", err)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return fmt.Errorf("base_url scheme must be http or https, got %q", parsed.Scheme)
	}

	if strings.TrimSpace(parsed.Host) == "" {
		return errors.New("base_url host is required")
	}

	if strings.TrimSpace(cfg.Token) == "" && strings.TrimSpace(cfg.TokenFile) == "" {
		return errors.New("token or token_file is required")
	}

	if cfg.Timeout.Duration <= 0 {
		return errors.New("timeout must be > 0")
	}

	return nil
}

func (o Output) ShouldCreateDirs() bool {
	if o.CreateDirs == nil {
		return true
	}

	return *o.CreateDirs
}

func (o Output) ShouldKeepSpoolFiles() bool {
	if o.KeepSpoolFiles == nil {
		return false
	}

	return *o.KeepSpoolFiles
}
