package render

import (
	"errors"
	"strings"

	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/storage"
)

type FinalReportInput struct {
	Task storage.FinalReportTask

	Timezone string

	Width         int
	LineSeparator string
}

func RenderFinalReport(input FinalReportInput) (string, error) {
	if input.Task.PrintJob.ID <= 0 {
		return "", errors.New("render final report: print_job.id must be > 0")
	}
	if input.Task.WakePlan.ID <= 0 {
		return "", errors.New("render final report: wake_plan.id must be > 0")
	}
	if input.Task.Snapshot == nil {
		return "", errors.New("render final report: snapshot is nil")
	}
	if !input.Task.Snapshot.IsReady() {
		return "", errors.New("render final report: snapshot is not ready")
	}

	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}

	wakePlan := input.Task.WakePlan
	snapshot := input.Task.Snapshot
	advice := input.Task.Advice

	b := NewBuilder(input.Width, input.LineSeparator)

	b.Title("WHOOP FINAL REPORT")

	b.KeyValue("DATE", FormatLocalDate(wakePlan.Date, timezone))
	b.KeyValue("WAKE", FormatLocalTime(wakePlan.WakeAt, timezone))
	b.KeyValue("SNAPSHOT", string(snapshot.DataState))

	b.Separator()

	b.Center("CORE METRICS")
	b.KeyValue("SLEEP", FormatIntPtr(snapshot.SleepScore, "%"))
	b.KeyValue("RECOVERY", FormatIntPtr(snapshot.RecoveryScore, "%"))
	b.KeyValue("STRAIN", FormatFloatPtr(snapshot.DayStrain, 1, ""))
	b.KeyValue("SLEEP NEED", FormatIntPtr(snapshot.SleepVsNeedPct, "%"))
	b.KeyValue("AWAKE", FormatMinutesPtr(snapshot.AwakeMinutes))

	b.Separator()

	b.Center("SLEEP DETAILS")
	b.KeyValue("TOTAL", FormatMinutesPtr(snapshot.SleepMinutes))
	b.KeyValue("NEEDED", FormatMinutesPtr(snapshot.SleepNeededMinutes))
	b.KeyValue("DEEP", FormatMinutesPtr(snapshot.DeepSleepMinutes))
	b.KeyValue("REM", FormatMinutesPtr(snapshot.REMSleepMinutes))
	b.KeyValue("RESTORE", FormatMinutesPtr(snapshot.RestorativeMinutes))
	b.KeyValue("EFFICIENCY", FormatFloatPtr(snapshot.SleepEfficiencyPct, 1, "%"))

	b.Separator()

	b.Center("BODY SIGNALS")
	b.KeyValue("HRV", FormatFloatPtr(snapshot.HRVRMSSDMS, 1, "ms"))
	b.KeyValue("RHR", FormatIntPtr(snapshot.RestingHeartRateBPM, "bpm"))
	b.KeyValue("SpO2", FormatFloatPtr(snapshot.SpO2Pct, 1, "%"))
	b.KeyValue("RESP", FormatFloatPtr(snapshot.RespiratoryRate, 1, "/min"))
	b.KeyValue("SKIN TEMP", FormatFloatPtr(snapshot.SkinTempCelsius, 1, "C"))

	if advice != nil && advice.IsReady() {
		b.Separator()

		b.Center("COACH")
		if strings.TrimSpace(advice.RenderedText) != "" {
			b.Text(advice.RenderedText)
		} else {
			b.Text("Coach промолчал. Видимо, сам офигел от утренних метрик.")
		}

		if strings.TrimSpace(advice.Motto) != "" {
			b.Blank()
			b.Center("MOTTO")
			b.Text(advice.Motto)
		}
	}

	b.Separator()

	b.Center("REPORT: READY")
	b.Center("PHONE: STILL OPTIONAL")
	b.Center("DAY: DEPLOYED")

	b.Separator()

	return b.String(), nil
}

type FallbackReportInput struct {
	Task storage.FinalReportTask

	Timezone string

	Width         int
	LineSeparator string
}

func RenderFallbackReport(input FallbackReportInput) (string, error) {
	if input.Task.PrintJob.ID <= 0 {
		return "", errors.New("render fallback report: print_job.id must be > 0")
	}
	if input.Task.WakePlan.ID <= 0 {
		return "", errors.New("render fallback report: wake_plan.id must be > 0")
	}

	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}

	wakePlan := input.Task.WakePlan

	b := NewBuilder(input.Width, input.LineSeparator)

	b.Title("WHOOP FALLBACK REPORT")

	b.KeyValue("DATE", FormatLocalDate(wakePlan.Date, timezone))
	b.KeyValue("WAKE", FormatLocalTime(wakePlan.WakeAt, timezone))
	b.KeyValue("DEADLINE", FormatLocalTime(wakePlan.FinalDeadlineAt, timezone))

	b.Separator()

	b.Center("DATA STATUS")

	if input.Task.Snapshot == nil {
		b.Text("WHOOP snapshot ещё не готов.")
	} else {
		b.KeyValue("SNAPSHOT", string(input.Task.Snapshot.DataState))
		b.Text("Snapshot есть, но финальная картина ещё не собралась нормально.")
	}

	if input.Task.Advice == nil {
		b.Text("Coach тоже пока молчит. Без данных он не маг, а просто мужик с принтером.")
	} else {
		b.KeyValue("ADVICE", string(input.Task.Advice.Status))
	}

	b.Separator()

	b.Center("TEMPORARY ORDER")
	b.Text("Пока финальный отчёт не готов.")

	b.Separator()

	b.Center("REPORT: DELAYED")
	b.Center("PANIC: FALSE")
	b.Center("WAIT: TRUE")

	b.Separator()

	return b.String(), nil
}
