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

	AuthToken string `mapstructure:"auth_token"`

	AuthTokenFile string `mapstructure:"auth_token_file"`

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
	token := strings.TrimSpace(g.AuthToken)
	if token != "" {
		return token, nil
	}

	tokenFile := strings.TrimSpace(g.AuthTokenFile)
	if tokenFile == "" {
		return "", fmt.Errorf("printergateway.auth_token or printergateway.auth_token_file is required")
	}

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("read printergateway auth token file %q: %w", tokenFile, err)
	}

	token = strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("printergateway auth token file %q is empty", tokenFile)
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
