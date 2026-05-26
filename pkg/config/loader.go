package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Validatable interface {
	Validate() error
}

type Options struct {
	Paths         []string
	Names         []string
	Type          string
	EnvPrefix     string
	OptionalFiles bool
}

func Load[T any](opts Options) (T, error) {
	var zero T
	var cfg T

	v := viper.New()

	configType := strings.TrimSpace(opts.Type)
	if configType == "" {
		configType = "yaml"
	}
	v.SetConfigType(configType)

	foundAny := false

	for _, name := range opts.Names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		fileViper := viper.New()
		fileViper.SetConfigType(configType)
		fileViper.SetConfigName(name)

		for _, p := range opts.Paths {
			p = strings.TrimSpace(p)
			if p != "" {
				fileViper.AddConfigPath(p)
			}
		}

		if err := fileViper.ReadInConfig(); err == nil {
			if err := v.MergeConfigMap(fileViper.AllSettings()); err != nil {
				return zero, fmt.Errorf("merge config %q: %w", name, err)
			}
			foundAny = true
		}
	}

	if !foundAny && !opts.OptionalFiles && len(opts.Names) > 0 {
		return zero, fmt.Errorf("config files not found in paths %v for names %v", opts.Paths, opts.Names)
	}

	if prefix := strings.TrimSpace(opts.EnvPrefix); prefix != "" {
		v.SetEnvPrefix(prefix)
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.Unmarshal(&cfg); err != nil {
		return zero, fmt.Errorf("unmarshal config: %w", err)
	}

	if validatable, ok := any(&cfg).(Validatable); ok {
		if err := validatable.Validate(); err != nil {
			return zero, fmt.Errorf("invalid config: %w", err)
		}
	}

	return cfg, nil
}

func MustLoad[T any](opts Options) *T {
	cfg, err := Load[T](opts)
	if err != nil {
		panic(err)
	}

	return &cfg
}
