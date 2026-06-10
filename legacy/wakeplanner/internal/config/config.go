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

	WakePlanner WakePlanner `yaml:"wakeplanner"`
	PMSet       PMSet       `yaml:"pmset"`
}

type Logger struct {
	Level string `yaml:"level"`

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

	Token     string `yaml:"token"`
	TokenFile string `yaml:"token_file"`

	Timeout Duration `yaml:"timeout"`
}

type WakePlanner struct {
	UserID int64 `yaml:"user_id"`

	Lookahead   Duration `yaml:"lookahead"`
	PreWakeLead Duration `yaml:"pre_wake_lead"`

	SleepAfterPlanning bool `yaml:"sleep_after_planning"`
	DryRun             bool `yaml:"dry_run"`
}

type PMSet struct {
	Path string `yaml:"path"`
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
		"./config/wakeplanner.yaml",
		"./wakeplanner.yaml",
		"./config.local.yaml",
		"/opt/whoop-morning-printer/config/wakeplanner.yaml",
		"/opt/whoop-morning-printer/config/config.local.yaml",
		"/etc/whoop-morning-printer/wakeplanner.yaml",
		"/etc/whoop-morning-printer/config.local.yaml",
	}
}

func (c *Config) setDefaults() {
	c.AppName = strings.TrimSpace(c.AppName)
	if c.AppName == "" {
		c.AppName = "wakeplanner-legacy"
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
		c.Logger.FilePath = "./logs/wakeplanner.log"
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

	if c.WakePlanner.Lookahead.Duration <= 0 {
		c.WakePlanner.Lookahead.Duration = 36 * time.Hour
	}

	if c.WakePlanner.PreWakeLead.Duration <= 0 {
		c.WakePlanner.PreWakeLead.Duration = 20 * time.Minute
	}

	if c.PMSet.Path = strings.TrimSpace(c.PMSet.Path); c.PMSet.Path == "" {
		c.PMSet.Path = "pmset"
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

	if c.WakePlanner.UserID <= 0 {
		return errors.New("wakeplanner.user_id must be > 0")
	}

	if c.WakePlanner.Lookahead.Duration <= 0 {
		return errors.New("wakeplanner.lookahead must be > 0")
	}

	if c.WakePlanner.PreWakeLead.Duration <= 0 {
		return errors.New("wakeplanner.pre_wake_lead must be > 0")
	}

	if strings.TrimSpace(c.PMSet.Path) == "" {
		return errors.New("pmset.path is required")
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
