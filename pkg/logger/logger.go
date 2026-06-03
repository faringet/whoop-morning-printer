package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Options struct {
	AppName     string
	Env         string
	Level       string
	JSON        bool
	FileEnabled bool
	FilePath    string
}

func NewLogger(opts Options) *slog.Logger {
	lvl := parseLevel(strings.ToLower(strings.TrimSpace(opts.Level)))
	handlerOpts := handlerOptions(lvl)

	w := writer(opts)

	var h slog.Handler
	if opts.JSON {
		h = slog.NewJSONHandler(w, handlerOpts)
	} else {
		h = slog.NewTextHandler(w, handlerOpts)
	}

	host, _ := os.Hostname()

	app := strings.TrimSpace(opts.AppName)
	if app == "" {
		app = "whoop-morning-printer"
	}

	env := strings.TrimSpace(opts.Env)
	if env == "" {
		env = "dev"
	}

	log := slog.New(h).With(
		slog.String("app", app),
		slog.String("env", env),
		slog.String("host", host),
		slog.Int("pid", os.Getpid()),
		slog.String("goarch", runtime.GOARCH),
		slog.String("goos", runtime.GOOS),
	)

	slog.SetDefault(log)

	return log
}

func writer(opts Options) io.Writer {
	console := os.Stdout

	if !opts.FileEnabled {
		return console
	}

	filePath := strings.TrimSpace(opts.FilePath)
	if filePath == "" {
		filePath = "./logs/app.log"
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "logger: create log dir failed: %v\n", err)
		return console
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: open log file failed: %v\n", err)
		return console
	}

	return io.MultiWriter(console, file)
}

func handlerOptions(lvl slog.Level) *slog.HandlerOptions {
	return &slog.HandlerOptions{
		Level:       lvl,
		AddSource:   lvl <= slog.LevelDebug,
		ReplaceAttr: normalizeCoreAttrs,
	}
}

func normalizeCoreAttrs(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey:
		if t, ok := a.Value.Any().(time.Time); ok {
			a.Value = slog.StringValue(t.UTC().Format(time.RFC3339Nano))
		}
		return a

	case slog.LevelKey:
		if lv, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(strings.ToLower(lv.String()))
		}
		return a

	default:
		if d, ok := a.Value.Any().(time.Duration); ok {
			a.Value = slog.StringValue(d.String())
			return a
		}
		return a
	}
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}
