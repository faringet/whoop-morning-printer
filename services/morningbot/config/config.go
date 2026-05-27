package config

import (
	"errors"
	"fmt"
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
