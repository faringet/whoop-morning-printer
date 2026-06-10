package planner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/logger"
	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/pmset"
	"github.com/faringet/whoop-morning-printer/legacy/wakeplanner/internal/storage"
)

const minWakeScheduleLead = 1 * time.Minute

type Config struct {
	UserID int64

	Lookahead   time.Duration
	PreWakeLead time.Duration

	SleepAfterPlanning bool
}

type Planner struct {
	log *logger.Logger
	cfg Config

	store storage.Store
	power PowerManager

	now func() time.Time
}

type PowerManager interface {
	ScheduleWakeOrPowerOn(ctx context.Context, wakeAt time.Time) (pmset.Result, error)
	SleepNow(ctx context.Context) (pmset.Result, error)
}

type Result struct {
	PlanFound bool

	WakePlanID int64
	WakeAt     time.Time

	ScheduledWakeAt time.Time

	ScheduleResult *pmset.Result
	SleepResult    *pmset.Result
}

func New(log *logger.Logger, cfg Config, store storage.Store, power PowerManager) (*Planner, error) {
	if store == nil {
		return nil, errors.New("wakeplanner planner: store is nil")
	}
	if power == nil {
		return nil, errors.New("wakeplanner planner: power manager is nil")
	}
	if cfg.UserID <= 0 {
		return nil, errors.New("wakeplanner planner: user_id must be > 0")
	}
	if cfg.Lookahead <= 0 {
		cfg.Lookahead = 36 * time.Hour
	}
	if cfg.PreWakeLead <= 0 {
		cfg.PreWakeLead = 20 * time.Minute
	}

	return &Planner{
		log:   log,
		cfg:   cfg,
		store: store,
		power: power,
		now:   time.Now,
	}, nil
}

func (p *Planner) RunOnce(ctx context.Context) (Result, error) {
	if p == nil {
		return Result{}, errors.New("wakeplanner planner: planner is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	now := p.now()

	if p.log != nil {
		p.log.Info("wake planning started",
			"user_id", p.cfg.UserID,
			"now_utc", now.UTC().Format(time.RFC3339),
			"now_local", now.Local().Format(time.RFC3339),
			"lookahead", p.cfg.Lookahead,
			"pre_wake_lead", p.cfg.PreWakeLead,
			"sleep_after_planning", p.cfg.SleepAfterPlanning,
		)
	}

	plan, err := p.store.GetNextWakePlan(ctx, storage.GetNextWakePlanInput{
		UserID:    p.cfg.UserID,
		Now:       now.UTC(),
		Lookahead: p.cfg.Lookahead,
	})
	if err != nil {
		return Result{}, fmt.Errorf("get next wake plan: %w", err)
	}

	if plan == nil {
		if p.log != nil {
			p.log.Info("no next wake plan found",
				"user_id", p.cfg.UserID,
				"lookahead", p.cfg.Lookahead,
			)
		}

		return Result{
			PlanFound: false,
		}, nil
	}

	if err := validateWakePlan(*plan); err != nil {
		return Result{}, fmt.Errorf("invalid wake plan: %w", err)
	}

	scheduledWakeAt := plan.WakeAt.Add(-p.cfg.PreWakeLead)

	if !scheduledWakeAt.After(now.Add(minWakeScheduleLead)) {
		return Result{}, fmt.Errorf(
			"computed wake time is too close or in the past: wake_at=%s pre_wake_lead=%s scheduled_wake_at=%s now=%s",
			plan.WakeAt.UTC().Format(time.RFC3339),
			p.cfg.PreWakeLead,
			scheduledWakeAt.UTC().Format(time.RFC3339),
			now.UTC().Format(time.RFC3339),
		)
	}

	if p.log != nil {
		p.log.Info("next wake plan selected",
			"wake_plan_id", plan.ID,
			"user_id", plan.UserID,
			"wake_at_utc", plan.WakeAt.UTC().Format(time.RFC3339),
			"wake_at_local", plan.WakeAt.Local().Format(time.RFC3339),
			"scheduled_wake_at_utc", scheduledWakeAt.UTC().Format(time.RFC3339),
			"scheduled_wake_at_local", scheduledWakeAt.Local().Format(time.RFC3339),
			"status", plan.Status,
		)
	}

	scheduleResult, err := p.power.ScheduleWakeOrPowerOn(ctx, scheduledWakeAt)
	if err != nil {
		return Result{}, fmt.Errorf("schedule wakeorpoweron: %w", err)
	}

	if p.log != nil {
		p.log.Info("wake event scheduled",
			"command", scheduleResult.Command,
			"args", strings.Join(scheduleResult.Args, " "),
			"dry_run", scheduleResult.DryRun,
			"output", scheduleResult.Output,
		)
	}

	result := Result{
		PlanFound: true,

		WakePlanID: plan.ID,
		WakeAt:     plan.WakeAt,

		ScheduledWakeAt: scheduledWakeAt,
		ScheduleResult:  &scheduleResult,
	}

	if !p.cfg.SleepAfterPlanning {
		if p.log != nil {
			p.log.Info("sleep after planning disabled")
		}

		return result, nil
	}

	sleepResult, err := p.power.SleepNow(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("sleep now: %w", err)
	}

	result.SleepResult = &sleepResult

	if p.log != nil {
		p.log.Info("sleepnow executed",
			"command", sleepResult.Command,
			"args", strings.Join(sleepResult.Args, " "),
			"dry_run", sleepResult.DryRun,
			"output", sleepResult.Output,
		)
	}

	return result, nil
}

func validateWakePlan(plan storage.WakePlan) error {
	if plan.ID <= 0 {
		return errors.New("id must be > 0")
	}
	if plan.UserID <= 0 {
		return errors.New("user_id must be > 0")
	}
	if !plan.HasWakeTime() {
		return errors.New("wake_at is required")
	}
	if !plan.IsScheduled() {
		return fmt.Errorf("status must be scheduled, got %q", plan.Status)
	}

	return nil
}
