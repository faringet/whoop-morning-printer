package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/morningbot/internal/storage"
)

var ErrWakeTimeInPast = errors.New("orchestrator: wake time is in the past")

type Config struct {
	UserID int64

	Timezone        string
	DefaultWakeTime string

	PrepareBefore      time.Duration
	FinalDeadlineAfter time.Duration
}

type Orchestrator struct {
	store storage.Store
	cfg   Config
	now   func() time.Time
}

type ScheduleWakeInput struct {
	Command        WakeCommand
	TelegramUserID *int64
}

type ScheduleWakeAtInput struct {
	WakeAt         time.Time
	TelegramUserID *int64
}

type StatusResult struct {
	WakePlan storage.WakePlan
}

type CancelResult struct {
	WakePlan storage.WakePlan
}

type TestPrintResult struct {
	PrintJob storage.PrintJob
}

func New(store storage.Store, cfg Config) (*Orchestrator, error) {
	if store == nil {
		return nil, errors.New("orchestrator: store is nil")
	}
	if cfg.UserID <= 0 {
		return nil, errors.New("orchestrator: user_id must be > 0")
	}

	cfg.Timezone = strings.TrimSpace(cfg.Timezone)
	if cfg.Timezone == "" {
		cfg.Timezone = "Europe/Moscow"
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return nil, fmt.Errorf("orchestrator: invalid timezone: %w", err)
	}

	if _, err := normalizeWakeTime(cfg.DefaultWakeTime); err != nil {
		return nil, fmt.Errorf("orchestrator: invalid default wake time: %w", err)
	}

	if cfg.PrepareBefore <= 0 {
		cfg.PrepareBefore = 5 * time.Minute
	}
	if cfg.FinalDeadlineAfter <= 0 {
		cfg.FinalDeadlineAfter = 90 * time.Minute
	}

	return &Orchestrator{
		store: store,
		cfg:   cfg,
		now:   time.Now,
	}, nil
}

func (o *Orchestrator) ScheduleWake(ctx context.Context, input ScheduleWakeInput) (storage.ScheduleWakePlanResult, error) {
	if o == nil {
		return storage.ScheduleWakePlanResult{}, errors.New("orchestrator: orchestrator is nil")
	}

	if strings.TrimSpace(input.Command.WakeTime) == "" {
		wakeTime, err := normalizeWakeTime(o.cfg.DefaultWakeTime)
		if err != nil {
			return storage.ScheduleWakePlanResult{}, err
		}

		input.Command.WakeTime = wakeTime
	}

	if input.Command.Day == "" {
		input.Command.Day = WakeDayAuto
	}

	if err := o.ensureUser(ctx, input.TelegramUserID); err != nil {
		return storage.ScheduleWakePlanResult{}, err
	}

	wakeAt, err := o.resolveWakeAt(input.Command)
	if err != nil {
		return storage.ScheduleWakePlanResult{}, err
	}

	return o.scheduleWakePlan(ctx, wakeAt)
}

func (o *Orchestrator) ScheduleWakeAt(ctx context.Context, input ScheduleWakeAtInput) (storage.ScheduleWakePlanResult, error) {
	if o == nil {
		return storage.ScheduleWakePlanResult{}, errors.New("orchestrator: orchestrator is nil")
	}

	if err := o.ensureUser(ctx, input.TelegramUserID); err != nil {
		return storage.ScheduleWakePlanResult{}, err
	}

	if input.WakeAt.IsZero() {
		return storage.ScheduleWakePlanResult{}, fmt.Errorf("%w: wake_at is required", ErrInvalidWakeTime)
	}

	wakeAt := input.WakeAt.UTC()
	if wakeAt.Before(o.now().UTC()) {
		return storage.ScheduleWakePlanResult{}, ErrWakeTimeInPast
	}

	return o.scheduleWakePlan(ctx, wakeAt)
}

func (o *Orchestrator) Status(ctx context.Context) (StatusResult, error) {
	if o == nil {
		return StatusResult{}, errors.New("orchestrator: orchestrator is nil")
	}

	wakePlan, err := o.store.GetNearestActiveWakePlan(ctx, o.cfg.UserID, o.now().UTC())
	if err != nil {
		return StatusResult{}, err
	}

	return StatusResult{
		WakePlan: wakePlan,
	}, nil
}

func (o *Orchestrator) Cancel(ctx context.Context) (CancelResult, error) {
	if o == nil {
		return CancelResult{}, errors.New("orchestrator: orchestrator is nil")
	}

	wakePlan, err := o.store.CancelNearestActiveWakePlan(ctx, o.cfg.UserID, o.now().UTC())
	if err != nil {
		return CancelResult{}, err
	}

	return CancelResult{
		WakePlan: wakePlan,
	}, nil
}

func (o *Orchestrator) CreateTestPrintJob(ctx context.Context) (TestPrintResult, error) {
	if o == nil {
		return TestPrintResult{}, errors.New("orchestrator: orchestrator is nil")
	}

	if err := o.ensureUser(ctx, nil); err != nil {
		return TestPrintResult{}, err
	}

	payload := strings.Join([]string{
		"WHOOP MORNING PRINTER",
		"TEST PRINT JOB",
		"",
		"Если этот текст доехал до print_jobs,",
		"значит бот жив, Postgres жив,",
		"а утренняя дисциплина уже где-то рядом.",
		"",
		"Стетхэм бы одобрил.",
	}, "\n")

	job, err := o.store.CreateTestPrintJob(ctx, storage.CreateTestPrintJobInput{
		UserID:      o.cfg.UserID,
		NotBefore:   o.now().UTC(),
		PayloadText: payload,
	})
	if err != nil {
		return TestPrintResult{}, err
	}

	return TestPrintResult{
		PrintJob: job,
	}, nil
}

func (o *Orchestrator) scheduleWakePlan(ctx context.Context, wakeAt time.Time) (storage.ScheduleWakePlanResult, error) {
	wakeAt = wakeAt.UTC()

	prepareAt := wakeAt.Add(-o.cfg.PrepareBefore)
	finalDeadlineAt := wakeAt.Add(o.cfg.FinalDeadlineAfter)

	return o.store.ScheduleWakePlan(ctx, storage.ScheduleWakePlanInput{
		UserID: o.cfg.UserID,

		Date:            dateOnlyUTC(wakeAt, o.cfg.Timezone),
		WakeAt:          wakeAt,
		PrepareAt:       prepareAt,
		FinalDeadlineAt: finalDeadlineAt,

		Source: storage.WakePlanSourceTelegram,
	})
}

func (o *Orchestrator) ensureUser(ctx context.Context, telegramUserID *int64) error {
	return o.store.EnsureUser(ctx, storage.EnsureUserInput{
		UserID:         o.cfg.UserID,
		TelegramUserID: telegramUserID,
		Timezone:       o.cfg.Timezone,
	})
}

func (o *Orchestrator) resolveWakeAt(command WakeCommand) (time.Time, error) {
	loc, err := time.LoadLocation(o.cfg.Timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("orchestrator: load timezone: %w", err)
	}

	wakeTime, err := normalizeWakeTime(command.WakeTime)
	if err != nil {
		return time.Time{}, err
	}

	parsed, err := time.Parse("15:04", wakeTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: expected HH:MM format", ErrInvalidWakeTime)
	}

	now := o.now().In(loc)
	year, month, day := now.Date()

	wakeAtToday := time.Date(
		year,
		month,
		day,
		parsed.Hour(),
		parsed.Minute(),
		0,
		0,
		loc,
	)

	switch command.Day {
	case WakeDayToday:
		if wakeAtToday.Before(now) {
			return time.Time{}, ErrWakeTimeInPast
		}

		return wakeAtToday.UTC(), nil

	case WakeDayTomorrow:
		return wakeAtToday.AddDate(0, 0, 1).UTC(), nil

	case WakeDayAuto, "":
		if wakeAtToday.After(now) {
			return wakeAtToday.UTC(), nil
		}

		return wakeAtToday.AddDate(0, 0, 1).UTC(), nil

	default:
		return time.Time{}, fmt.Errorf("orchestrator: unsupported wake day %q", command.Day)
	}
}

func dateOnlyUTC(t time.Time, timezone string) time.Time {
	loc := time.UTC

	timezone = strings.TrimSpace(timezone)
	if timezone != "" {
		if loaded, err := time.LoadLocation(timezone); err == nil {
			loc = loaded
		}
	}

	year, month, day := t.In(loc).Date()

	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
}
