package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Base struct {
	AppName string `mapstructure:"app_name"`
	Env     string `mapstructure:"env"`
}

func (b *Base) Validate() error {
	if b == nil {
		return errors.New("base config is nil")
	}

	b.AppName = strings.TrimSpace(b.AppName)
	b.Env = strings.TrimSpace(strings.ToLower(b.Env))

	return nil
}

type Runtime struct {
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

func (r *Runtime) Validate() error {
	if r == nil {
		return errors.New("runtime config is nil")
	}
	if r.ShutdownTimeout < 0 {
		return errors.New("runtime.shutdown_timeout must be >= 0")
	}
	return nil
}

type Logger struct {
	Level       string `mapstructure:"level"`
	JSON        bool   `mapstructure:"json"`
	FileEnabled bool   `mapstructure:"file_enabled"`
	FilePath    string `mapstructure:"file_path"`
}

func (l *Logger) Validate() error {
	if l == nil {
		return errors.New("logger config is nil")
	}

	l.Level = strings.ToLower(strings.TrimSpace(l.Level))
	if l.Level == "" {
		return errors.New("logger.level is required")
	}

	switch l.Level {
	case "debug", "info", "warn", "warning", "error":
	default:
		return fmt.Errorf("logger.level must be one of [debug, info, warn, warning, error], got %q", l.Level)
	}

	l.FilePath = strings.TrimSpace(l.FilePath)

	if l.FileEnabled && l.FilePath == "" {
		return errors.New("logger.file_path is required when logger.file_enabled=true")
	}

	return nil
}

type Storage struct {
	Postgres PostgresStorage `mapstructure:"postgres"`
}

func (s *Storage) Validate() error {
	if s == nil {
		return errors.New("storage config is nil")
	}
	return s.Postgres.Validate()
}

type PostgresStorage struct {
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

func (p *PostgresStorage) Validate() error {
	if p == nil {
		return errors.New("storage.postgres config is nil")
	}

	p.DSN = strings.TrimSpace(p.DSN)
	if p.DSN == "" {
		return errors.New("storage.postgres.dsn is required")
	}
	if p.MaxOpenConns < 0 {
		return errors.New("storage.postgres.max_open_conns must be >= 0")
	}
	if p.MaxIdleConns < 0 {
		return errors.New("storage.postgres.max_idle_conns must be >= 0")
	}
	if p.ConnMaxLifetime < 0 {
		return errors.New("storage.postgres.conn_max_lifetime must be >= 0")
	}
	if p.ConnMaxIdleTime < 0 {
		return errors.New("storage.postgres.conn_max_idle_time must be >= 0")
	}

	return nil
}

type Ollama struct {
	BaseURL   string        `mapstructure:"base_url"`
	Timeout   time.Duration `mapstructure:"timeout"`
	Model     string        `mapstructure:"model"`
	KeepAlive string        `mapstructure:"keep_alive"`
}

func (o *Ollama) Validate(enabled bool) error {
	if !enabled {
		return nil
	}
	if o == nil {
		return errors.New("ollama config is nil")
	}

	o.BaseURL = strings.TrimSpace(o.BaseURL)
	o.Model = strings.TrimSpace(o.Model)
	o.KeepAlive = strings.TrimSpace(o.KeepAlive)

	if o.BaseURL == "" {
		return errors.New("ollama.base_url is required")
	}
	if o.Model == "" {
		return errors.New("ollama.model is required")
	}
	if o.Timeout <= 0 {
		return errors.New("ollama.timeout must be > 0")
	}

	return nil
}

type TelegramBot struct {
	Token       string        `mapstructure:"token"`
	Debug       bool          `mapstructure:"debug"`
	UseWebhook  bool          `mapstructure:"use_webhook"`
	WebhookURL  string        `mapstructure:"webhook_url"`
	PollTimeout time.Duration `mapstructure:"poll_timeout"`
}

func (t *TelegramBot) Validate(enabled bool) error {
	if !enabled {
		return nil
	}
	if t == nil {
		return errors.New("telegram_bot config is nil")
	}

	t.Token = strings.TrimSpace(t.Token)
	t.WebhookURL = strings.TrimSpace(t.WebhookURL)

	if t.Token == "" {
		return errors.New("telegram_bot.token is required")
	}
	if t.UseWebhook && t.WebhookURL == "" {
		return errors.New("telegram_bot.webhook_url is required when use_webhook=true")
	}
	if !t.UseWebhook && t.PollTimeout <= 0 {
		return errors.New("telegram_bot.poll_timeout must be > 0 when use_webhook=false")
	}

	return nil
}
