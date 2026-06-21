package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	pcfg "github.com/faringet/whoop-morning-printer/pkg/config"
)

type Config struct {
	Base    pcfg.Base    `mapstructure:",squash"`
	Logger  pcfg.Logger  `mapstructure:"logger"`
	Runtime pcfg.Runtime `mapstructure:"runtime"`
	Storage pcfg.Storage `mapstructure:"storage"`

	PrinterGateway PrinterGateway `mapstructure:"printergateway"`
}

type PrinterGateway struct {
	HTTPAddr string `mapstructure:"http_addr"`

	AuthToken     string `mapstructure:"auth_token"`
	AuthTokenFile string `mapstructure:"auth_token_file"`

	DisplayAuthToken     string        `mapstructure:"display_auth_token"`
	DisplayAuthTokenFile string        `mapstructure:"display_auth_token_file"`
	DisplayUserID        int64         `mapstructure:"display_user_id"`
	DisplayTimezone      string        `mapstructure:"display_timezone"`
	DisplayLookahead     time.Duration `mapstructure:"display_lookahead"`

	StartupTimeout time.Duration `mapstructure:"startup_timeout"`
	PingInterval   time.Duration `mapstructure:"ping_interval"`

	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
}

func (c *Config) setDefaults() {
	if c.Base.AppName == "" {
		c.Base.AppName = "printergateway"
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
		c.Storage.Postgres.MaxOpenConns = 8
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

	c.PrinterGateway.HTTPAddr = strings.TrimSpace(c.PrinterGateway.HTTPAddr)
	if c.PrinterGateway.HTTPAddr == "" {
		c.PrinterGateway.HTTPAddr = ":8088"
	}

	c.PrinterGateway.AuthToken = strings.TrimSpace(c.PrinterGateway.AuthToken)
	c.PrinterGateway.AuthTokenFile = strings.TrimSpace(c.PrinterGateway.AuthTokenFile)

	c.PrinterGateway.DisplayAuthToken = strings.TrimSpace(c.PrinterGateway.DisplayAuthToken)
	c.PrinterGateway.DisplayAuthTokenFile = strings.TrimSpace(c.PrinterGateway.DisplayAuthTokenFile)

	if c.PrinterGateway.DisplayUserID <= 0 {
		c.PrinterGateway.DisplayUserID = 1
	}

	c.PrinterGateway.DisplayTimezone = strings.TrimSpace(c.PrinterGateway.DisplayTimezone)
	if c.PrinterGateway.DisplayTimezone == "" {
		c.PrinterGateway.DisplayTimezone = "Europe/Moscow"
	}

	if c.PrinterGateway.DisplayLookahead <= 0 {
		c.PrinterGateway.DisplayLookahead = 36 * time.Hour
	}

	if c.PrinterGateway.StartupTimeout <= 0 {
		c.PrinterGateway.StartupTimeout = 60 * time.Second
	}
	if c.PrinterGateway.PingInterval <= 0 {
		c.PrinterGateway.PingInterval = 2 * time.Second
	}

	if c.PrinterGateway.ReadHeaderTimeout <= 0 {
		c.PrinterGateway.ReadHeaderTimeout = 5 * time.Second
	}
	if c.PrinterGateway.ReadTimeout <= 0 {
		c.PrinterGateway.ReadTimeout = 15 * time.Second
	}
	if c.PrinterGateway.WriteTimeout <= 0 {
		c.PrinterGateway.WriteTimeout = 15 * time.Second
	}
	if c.PrinterGateway.IdleTimeout <= 0 {
		c.PrinterGateway.IdleTimeout = 60 * time.Second
	}
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

	if c.PrinterGateway.HTTPAddr == "" {
		return fmt.Errorf("printergateway.http_addr is required")
	}
	if c.PrinterGateway.AuthToken == "" && c.PrinterGateway.AuthTokenFile == "" {
		return fmt.Errorf("printergateway.auth_token or printergateway.auth_token_file is required")
	}
	if c.PrinterGateway.DisplayAuthToken == "" && c.PrinterGateway.DisplayAuthTokenFile == "" {
		return fmt.Errorf("printergateway.display_auth_token or printergateway.display_auth_token_file is required")
	}
	if c.PrinterGateway.DisplayUserID <= 0 {
		return fmt.Errorf("printergateway.display_user_id must be > 0")
	}
	if _, err := time.LoadLocation(c.PrinterGateway.DisplayTimezone); err != nil {
		return fmt.Errorf("printergateway.display_timezone is invalid: %w", err)
	}
	if c.PrinterGateway.DisplayLookahead <= 0 {
		return fmt.Errorf("printergateway.display_lookahead must be > 0")
	}
	if c.PrinterGateway.StartupTimeout <= 0 {
		return fmt.Errorf("printergateway.startup_timeout must be > 0")
	}
	if c.PrinterGateway.PingInterval <= 0 {
		return fmt.Errorf("printergateway.ping_interval must be > 0")
	}
	if c.PrinterGateway.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("printergateway.read_header_timeout must be > 0")
	}
	if c.PrinterGateway.ReadTimeout <= 0 {
		return fmt.Errorf("printergateway.read_timeout must be > 0")
	}
	if c.PrinterGateway.WriteTimeout <= 0 {
		return fmt.Errorf("printergateway.write_timeout must be > 0")
	}
	if c.PrinterGateway.IdleTimeout <= 0 {
		return fmt.Errorf("printergateway.idle_timeout must be > 0")
	}

	return nil
}

func (g PrinterGateway) AuthTokenValue() (string, error) {
	return tokenValue(
		"printergateway.auth_token",
		"printergateway.auth_token_file",
		g.AuthToken,
		g.AuthTokenFile,
	)
}

func (g PrinterGateway) DisplayAuthTokenValue() (string, error) {
	return tokenValue(
		"printergateway.display_auth_token",
		"printergateway.display_auth_token_file",
		g.DisplayAuthToken,
		g.DisplayAuthTokenFile,
	)
}

func tokenValue(
	tokenName string,
	tokenFileName string,
	tokenValue string,
	tokenFileValue string,
) (string, error) {
	token := strings.TrimSpace(tokenValue)
	if token != "" {
		return token, nil
	}

	tokenFile := strings.TrimSpace(tokenFileValue)
	if tokenFile == "" {
		return "", fmt.Errorf("%s or %s is required", tokenName, tokenFileName)
	}

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("read %s %q: %w", tokenFileName, tokenFile, err)
	}

	token = strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("%s %q is empty", tokenFileName, tokenFile)
	}

	return token, nil
}

func New() *Config {
	c := pcfg.MustLoad[Config](pcfg.Options{
		Paths: []string{
			"./services/printergateway/config",
			"./config",
			"./configs",
			"/etc/whoop-morning-printer",
		},
		Names:         []string{"printergateway", "config", "config.local"},
		Type:          "yaml",
		EnvPrefix:     "PRINTERGATEWAY",
		OptionalFiles: true,
	})

	if err := c.Validate(); err != nil {
		panic(fmt.Errorf("invalid printergateway config: %w", err))
	}

	return c
}
