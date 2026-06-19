package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	pcfg "github.com/faringet/whoop-morning-printer/pkg/config"
)

type Config struct {
	Base        pcfg.Base        `mapstructure:",squash"`
	Logger      pcfg.Logger      `mapstructure:"logger"`
	Runtime     pcfg.Runtime     `mapstructure:"runtime"`
	Storage     pcfg.Storage     `mapstructure:"storage"`
	TelegramBot pcfg.TelegramBot `mapstructure:"telegram_bot"`

	Access     Access     `mapstructure:"access"`
	MorningBot MorningBot `mapstructure:"morningbot"`
	HTTP       HTTP       `mapstructure:"http"`
	MiniApp    MiniApp    `mapstructure:"mini_app"`
}

type Access struct {
	AllowedUserIDs []int64 `mapstructure:"allowed_user_ids"`
	AllowedChatIDs []int64 `mapstructure:"allowed_chat_ids"`
	DenyMessage    string  `mapstructure:"deny_message"`
}

func (a *Access) setDefaults() {
	if strings.TrimSpace(a.DenyMessage) == "" {
		a.DenyMessage = "⛔ Доступ к боту ограничен."
	}
}

func (a *Access) Validate() error {
	if a == nil {
		return errors.New("access config is nil")
	}

	seenUsers := make(map[int64]struct{}, len(a.AllowedUserIDs))
	for _, id := range a.AllowedUserIDs {
		if id == 0 {
			return errors.New("access.allowed_user_ids must not contain 0")
		}
		if _, ok := seenUsers[id]; ok {
			return fmt.Errorf("duplicate allowed user id: %d", id)
		}
		seenUsers[id] = struct{}{}
	}

	seenChats := make(map[int64]struct{}, len(a.AllowedChatIDs))
	for _, id := range a.AllowedChatIDs {
		if id == 0 {
			return errors.New("access.allowed_chat_ids must not contain 0")
		}
		if _, ok := seenChats[id]; ok {
			return fmt.Errorf("duplicate allowed chat id: %d", id)
		}
		seenChats[id] = struct{}{}
	}

	return nil
}

type MorningBot struct {
	UserID int64 `mapstructure:"user_id"`

	Timezone        string `mapstructure:"timezone"`
	DefaultWakeTime string `mapstructure:"default_wake_time"`

	PrepareBefore      time.Duration `mapstructure:"prepare_before"`
	FinalDeadlineAfter time.Duration `mapstructure:"final_deadline_after"`
}

func (m *MorningBot) setDefaults() {
	m.Timezone = strings.TrimSpace(m.Timezone)
	if m.Timezone == "" {
		m.Timezone = "Europe/Moscow"
	}

	m.DefaultWakeTime = strings.TrimSpace(m.DefaultWakeTime)
	if m.DefaultWakeTime == "" {
		m.DefaultWakeTime = "08:30"
	}

	if m.PrepareBefore <= 0 {
		m.PrepareBefore = 5 * time.Minute
	}
	if m.FinalDeadlineAfter <= 0 {
		m.FinalDeadlineAfter = 90 * time.Minute
	}
}

func (m *MorningBot) Validate() error {
	if m == nil {
		return errors.New("morningbot config is nil")
	}

	if m.UserID <= 0 {
		return errors.New("morningbot.user_id must be > 0")
	}
	if _, err := time.LoadLocation(m.Timezone); err != nil {
		return fmt.Errorf("morningbot.timezone is invalid: %w", err)
	}
	if err := validateWakeTime(m.DefaultWakeTime); err != nil {
		return fmt.Errorf("morningbot.default_wake_time is invalid: %w", err)
	}
	if m.PrepareBefore <= 0 {
		return errors.New("morningbot.prepare_before must be > 0")
	}
	if m.FinalDeadlineAfter <= 0 {
		return errors.New("morningbot.final_deadline_after must be > 0")
	}

	return nil
}

type HTTP struct {
	Enabled bool   `mapstructure:"enabled"`
	Addr    string `mapstructure:"addr"`

	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`

	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

func (h *HTTP) setDefaults() {
	h.Addr = strings.TrimSpace(h.Addr)
	if h.Addr == "" {
		h.Addr = "127.0.0.1:8086"
	}

	if h.ReadHeaderTimeout <= 0 {
		h.ReadHeaderTimeout = 5 * time.Second
	}
	if h.ReadTimeout <= 0 {
		h.ReadTimeout = 10 * time.Second
	}
	if h.WriteTimeout <= 0 {
		h.WriteTimeout = 10 * time.Second
	}
	if h.IdleTimeout <= 0 {
		h.IdleTimeout = 60 * time.Second
	}

	for i := range h.AllowedOrigins {
		h.AllowedOrigins[i] = strings.TrimSpace(h.AllowedOrigins[i])
	}
}

func (h *HTTP) Validate() error {
	if h == nil {
		return errors.New("http config is nil")
	}
	if !h.Enabled {
		return nil
	}

	host, portValue, err := net.SplitHostPort(h.Addr)
	if err != nil {
		return fmt.Errorf("http.addr is invalid: %w", err)
	}

	port, err := strconv.Atoi(portValue)
	if err != nil || port <= 0 || port > 65535 {
		return fmt.Errorf("http.addr contains invalid port: %q", portValue)
	}

	if strings.Contains(host, "/") {
		return errors.New("http.addr contains invalid host")
	}
	if h.ReadHeaderTimeout <= 0 {
		return errors.New("http.read_header_timeout must be > 0")
	}
	if h.ReadTimeout <= 0 {
		return errors.New("http.read_timeout must be > 0")
	}
	if h.WriteTimeout <= 0 {
		return errors.New("http.write_timeout must be > 0")
	}
	if h.IdleTimeout <= 0 {
		return errors.New("http.idle_timeout must be > 0")
	}

	seenOrigins := make(map[string]struct{}, len(h.AllowedOrigins))
	for _, origin := range h.AllowedOrigins {
		if err := validateAllowedOrigin(origin); err != nil {
			return err
		}
		if _, ok := seenOrigins[origin]; ok {
			return fmt.Errorf("duplicate http.allowed_origins value: %q", origin)
		}
		seenOrigins[origin] = struct{}{}
	}

	return nil
}

type MiniApp struct {
	AuthMaxAge time.Duration `mapstructure:"auth_max_age"`
	DevAuth    DevAuth       `mapstructure:"dev_auth"`
}

type DevAuth struct {
	Enabled        bool  `mapstructure:"enabled"`
	TelegramUserID int64 `mapstructure:"telegram_user_id"`
}

func (m *MiniApp) setDefaults() {
	if m.AuthMaxAge <= 0 {
		m.AuthMaxAge = 10 * time.Minute
	}
}

func (m *MiniApp) Validate() error {
	if m == nil {
		return errors.New("mini_app config is nil")
	}
	if m.AuthMaxAge <= 0 {
		return errors.New("mini_app.auth_max_age must be > 0")
	}
	if m.DevAuth.Enabled && m.DevAuth.TelegramUserID <= 0 {
		return errors.New("mini_app.dev_auth.telegram_user_id must be > 0")
	}

	return nil
}

func (c *Config) setDefaults() {
	if c.Base.AppName == "" {
		c.Base.AppName = "morningbot"
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

	if c.TelegramBot.PollTimeout <= 0 {
		c.TelegramBot.PollTimeout = 30 * time.Second
	}

	c.Access.setDefaults()
	c.MorningBot.setDefaults()
	c.HTTP.setDefaults()
	c.MiniApp.setDefaults()
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
	if err := c.TelegramBot.Validate(true); err != nil {
		return fmt.Errorf("telegram_bot: %w", err)
	}
	if c.TelegramBot.UseWebhook {
		return errors.New("telegram_bot.use_webhook is not supported yet, use polling mode")
	}
	if err := c.Access.Validate(); err != nil {
		return fmt.Errorf("access: %w", err)
	}
	if err := c.MorningBot.Validate(); err != nil {
		return fmt.Errorf("morningbot: %w", err)
	}
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http: %w", err)
	}
	if err := c.MiniApp.Validate(); err != nil {
		return fmt.Errorf("mini_app: %w", err)
	}

	if c.HTTP.Enabled && len(c.Access.AllowedUserIDs) == 0 {
		return errors.New("access.allowed_user_ids must not be empty when http.enabled is true")
	}

	if c.MiniApp.DevAuth.Enabled {
		if !c.HTTP.Enabled {
			return errors.New("mini_app.dev_auth.enabled requires http.enabled")
		}
		if !strings.EqualFold(strings.TrimSpace(c.Base.Env), "dev") {
			return errors.New("mini_app.dev_auth.enabled is allowed only in dev environment")
		}
		if !isLoopbackAddress(c.HTTP.Addr) {
			return errors.New("mini_app.dev_auth.enabled requires http.addr to use a loopback address")
		}
		if !containsInt64(c.Access.AllowedUserIDs, c.MiniApp.DevAuth.TelegramUserID) {
			return errors.New("mini_app.dev_auth.telegram_user_id must be present in access.allowed_user_ids")
		}
	}

	return nil
}

func New() *Config {
	c := pcfg.MustLoad[Config](pcfg.Options{
		Paths: []string{
			"./services/morningbot/config",
			"./config",
			"./configs",
			"/etc/whoop-morning-printer",
		},
		Names:         []string{"morningbot", "config", "config.local"},
		Type:          "yaml",
		EnvPrefix:     "MORNINGBOT",
		OptionalFiles: true,
	})

	if err := c.Validate(); err != nil {
		panic(fmt.Errorf("invalid morningbot config: %w", err))
	}

	return c
}

func validateWakeTime(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("wake time is required")
	}

	if _, err := time.Parse("15:04", value); err != nil {
		return fmt.Errorf("expected HH:MM format: %w", err)
	}

	return nil
}

func validateAllowedOrigin(value string) error {
	if value == "" {
		return errors.New("http.allowed_origins must not contain empty values")
	}

	origin, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid http.allowed_origins value %q: %w", value, err)
	}
	if origin.Scheme != "http" && origin.Scheme != "https" {
		return fmt.Errorf("http.allowed_origins value %q must use http or https", value)
	}
	if origin.Host == "" {
		return fmt.Errorf("http.allowed_origins value %q must contain a host", value)
	}
	if origin.User != nil {
		return fmt.Errorf("http.allowed_origins value %q must not contain user info", value)
	}
	if origin.Path != "" && origin.Path != "/" {
		return fmt.Errorf("http.allowed_origins value %q must not contain a path", value)
	}
	if origin.RawQuery != "" || origin.Fragment != "" {
		return fmt.Errorf("http.allowed_origins value %q must not contain query or fragment", value)
	}

	return nil
}

func isLoopbackAddress(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}

	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

func containsInt64(items []int64, value int64) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}

	return false
}
