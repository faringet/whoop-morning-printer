package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Options struct {
	Level       string
	FileEnabled bool
	FilePath    string
}

type Logger struct {
	mu sync.Mutex

	level Level

	out  io.Writer
	file *os.File
}

func New(opts Options) (*Logger, error) {
	level := parseLevel(opts.Level)

	writers := []io.Writer{os.Stdout}

	var file *os.File

	if opts.FileEnabled {
		filePath := strings.TrimSpace(opts.FilePath)
		if filePath == "" {
			filePath = "./logs/wakeplanner.log"
		}

		if err := ensureDirForFile(filePath); err != nil {
			return nil, fmt.Errorf("create log dir: %w", err)
		}

		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("open log file %q: %w", filePath, err)
		}

		file = f
		writers = append(writers, f)
	}

	return &Logger{
		level: level,
		out:   io.MultiWriter(writers...),
		file:  file,
	}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}

	return l.file.Close()
}

func (l *Logger) Debug(msg string, kv ...interface{}) {
	l.write(LevelDebug, "debug", msg, kv...)
}

func (l *Logger) Info(msg string, kv ...interface{}) {
	l.write(LevelInfo, "info", msg, kv...)
}

func (l *Logger) Warn(msg string, kv ...interface{}) {
	l.write(LevelWarn, "warn", msg, kv...)
}

func (l *Logger) Error(msg string, kv ...interface{}) {
	l.write(LevelError, "error", msg, kv...)
}

func (l *Logger) write(level Level, levelName string, msg string, kv ...interface{}) {
	if l == nil {
		return
	}

	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().UTC().Format(time.RFC3339)

	fmt.Fprintf(l.out, "%s %-5s %s", timestamp, strings.ToUpper(levelName), strings.TrimSpace(msg))

	for i := 0; i < len(kv); i += 2 {
		key := fmt.Sprint(kv[i])

		var value interface{} = "<missing>"
		if i+1 < len(kv) {
			value = kv[i+1]
		}

		fmt.Fprintf(l.out, " %s=%q", sanitizeKey(key), fmt.Sprint(value))
	}

	fmt.Fprintln(l.out)
}

func parseLevel(raw string) Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "info", "":
		return LevelInfo
	default:
		return LevelInfo
	}
}

func sanitizeKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "field"
	}

	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "\t", "_")
	key = strings.ReplaceAll(key, "\n", "_")
	key = strings.ReplaceAll(key, "\r", "_")

	return key
}

func ensureDirForFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}
