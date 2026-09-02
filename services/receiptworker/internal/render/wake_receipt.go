package render

import (
	"errors"
	"strings"

	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/storage"
)

type WakeReceiptInput struct {
	Task storage.WakeReceiptTask

	Timezone string

	Width         int
	LineSeparator string

	ArtText   string
	FieldNote string
}

func RenderWakeReceipt(input WakeReceiptInput) (string, error) {
	if input.Task.PrintJob.ID <= 0 {
		return "", errors.New("render wake receipt: print_job.id must be > 0")
	}
	if input.Task.WakePlan.ID <= 0 {
		return "", errors.New("render wake receipt: wake_plan.id must be > 0")
	}

	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}

	b := NewBuilder(input.Width, input.LineSeparator)

	wakePlan := input.Task.WakePlan
	fieldNote := strings.TrimSpace(input.FieldNote)

	b.Title("WHOOP MORNING PRINTER")
	b.Center("WAKE RECEIPT")
	b.Blank()

	if strings.TrimSpace(input.ArtText) != "" {
		b.Raw(input.ArtText)
		b.Blank()
	}

	b.KeyValue("DATE", FormatLocalDate(wakePlan.Date, timezone))
	b.KeyValue("WAKE", FormatLocalTime(wakePlan.WakeAt, timezone))
	b.KeyValue("FINAL REPORT", FormatLocalTime(wakePlan.FinalDeadlineAt, timezone))

	b.Separator()

	b.Center("DAILY QUESTS")
	b.Blank()

	b.Line("[ ] PHONE-FREE WAKE")

	b.Blank()
	b.Blank()

	b.Line("[ ] LEETCODE")
	b.Line("[ ] ONE HARD THING")
	b.Line("[ ] TECH READING")
	b.Line("[ ] SOCIALS UNDER CONTROL")
	b.Line("[ ] ___H DEEP WORK")
	b.Line("[ ] DO SOMETHING OFFLINE")

	b.Blank()
	b.Blank()

	b.Line("[ ] MAIN QUEST: __________")

	b.Separator()

	if fieldNote != "" {
		b.Center("FIELD NOTE")
		b.Blank()
		b.Text(fieldNote)

		b.Separator()
	}

	b.Center("MAKE THE DAY COUNT.")

	return b.String(), nil
}
