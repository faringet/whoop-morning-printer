package pmset

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultPath = "pmset"

type Config struct {
	Path   string
	DryRun bool
}

type Client struct {
	path   string
	dryRun bool
}

type Result struct {
	Command string
	Args    []string

	DryRun bool
	Output string
}

func New(cfg Config) (*Client, error) {
	path := strings.TrimSpace(cfg.Path)
	if path == "" {
		path = defaultPath
	}

	return &Client{
		path:   path,
		dryRun: cfg.DryRun,
	}, nil
}

func (c *Client) ScheduleWakeOrPowerOn(ctx context.Context, wakeAt time.Time) (Result, error) {
	if c == nil {
		return Result{}, errors.New("wakeplanner pmset: client is nil")
	}
	if wakeAt.IsZero() {
		return Result{}, errors.New("wakeplanner pmset: wake_at is required")
	}

	localWakeAt := wakeAt.Local()
	formattedWakeAt := formatPMSetTime(localWakeAt)

	return c.run(ctx, "schedule", "wakeorpoweron", formattedWakeAt)
}

func (c *Client) SleepNow(ctx context.Context) (Result, error) {
	if c == nil {
		return Result{}, errors.New("wakeplanner pmset: client is nil")
	}

	return c.run(ctx, "sleepnow")
}

func (c *Client) run(ctx context.Context, args ...string) (Result, error) {
	if c == nil {
		return Result{}, errors.New("wakeplanner pmset: client is nil")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	result := Result{
		Command: c.path,
		Args:    append([]string(nil), args...),
		DryRun:  c.dryRun,
	}

	if c.dryRun {
		result.Output = "dry run: command not executed"
		return result, nil
	}

	cmd := exec.CommandContext(ctx, c.path, args...)

	output, err := cmd.CombinedOutput()
	result.Output = strings.TrimSpace(string(output))

	if err != nil {
		return result, fmt.Errorf("run %s %s: %w: %s", c.path, strings.Join(args, " "), err, result.Output)
	}

	return result, nil
}

func formatPMSetTime(t time.Time) string {
	return t.Format("01/02/06 15:04:05")
}
