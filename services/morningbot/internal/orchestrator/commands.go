package orchestrator

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidWakeArgs = errors.New("orchestrator: invalid wake arguments")
	ErrInvalidWakeTime = errors.New("orchestrator: invalid wake time")
)

type WakeDay string

const (
	WakeDayAuto     WakeDay = "auto"
	WakeDayToday    WakeDay = "today"
	WakeDayTomorrow WakeDay = "tomorrow"
)

type WakeCommand struct {
	Day      WakeDay
	WakeTime string
}

func ParseWakeArgs(args string, defaultWakeTime string) (WakeCommand, error) {
	defaultWakeTime, err := normalizeWakeTime(defaultWakeTime)
	if err != nil {
		return WakeCommand{}, fmt.Errorf("default wake time: %w", err)
	}

	fields := strings.Fields(strings.TrimSpace(args))
	if len(fields) == 0 {
		return WakeCommand{
			Day:      WakeDayAuto,
			WakeTime: defaultWakeTime,
		}, nil
	}

	if len(fields) > 2 {
		return WakeCommand{}, fmt.Errorf("%w: expected /wake HH:MM or /wake tomorrow HH:MM", ErrInvalidWakeArgs)
	}

	if len(fields) == 1 {
		if day, ok := parseWakeDay(fields[0]); ok {
			return WakeCommand{
				Day:      day,
				WakeTime: defaultWakeTime,
			}, nil
		}

		wakeTime, err := normalizeWakeTime(fields[0])
		if err != nil {
			return WakeCommand{}, err
		}

		return WakeCommand{
			Day:      WakeDayAuto,
			WakeTime: wakeTime,
		}, nil
	}

	first := fields[0]
	second := fields[1]

	if day, ok := parseWakeDay(first); ok {
		wakeTime, err := normalizeWakeTime(second)
		if err != nil {
			return WakeCommand{}, err
		}

		return WakeCommand{
			Day:      day,
			WakeTime: wakeTime,
		}, nil
	}

	if day, ok := parseWakeDay(second); ok {
		wakeTime, err := normalizeWakeTime(first)
		if err != nil {
			return WakeCommand{}, err
		}

		return WakeCommand{
			Day:      day,
			WakeTime: wakeTime,
		}, nil
	}

	return WakeCommand{}, fmt.Errorf("%w: expected day marker today/tomorrow or wake time HH:MM", ErrInvalidWakeArgs)
}

func parseWakeDay(value string) (WakeDay, bool) {
	value = strings.TrimSpace(strings.ToLower(value))

	switch value {
	case "today", "сегодня":
		return WakeDayToday, true

	case "tomorrow", "tmr", "завтра":
		return WakeDayTomorrow, true

	default:
		return "", false
	}
}

func normalizeWakeTime(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: empty wake time", ErrInvalidWakeTime)
	}

	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return "", fmt.Errorf("%w: expected HH:MM format", ErrInvalidWakeTime)
	}

	return parsed.Format("15:04"), nil
}
