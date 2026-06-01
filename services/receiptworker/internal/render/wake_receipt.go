package render

import (
	"errors"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/faringet/whoop-morning-printer/services/receiptworker/internal/storage"
)

type WakeReceiptInput struct {
	Task storage.WakeReceiptTask

	Timezone string

	Width         int
	LineSeparator string

	ArtName string
	ArtText string
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
	command := pickWakeCommand(wakePlan)
	rule := pickMorningRule(wakePlan)

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

	if strings.TrimSpace(input.ArtName) != "" {
		b.KeyValue("ART", input.ArtName)
	}

	b.Separator()

	b.Center("BOOT SEQUENCE")
	b.Line("01. ВСТАТЬ С КРОВАТИ")
	b.Line("02. НЕ ТРОГАТЬ ТЕЛЕФОН")
	b.Line("03. ВКЛЮЧИТЬ СВЕТ")
	b.Line("04. ДОЙТИ ДО ВАННОЙ")
	b.Line("05. ЖДАТЬ ВТОРОЙ ЧЕК")

	b.Separator()

	b.Center("MORNING ORDER")
	b.Text(command)

	b.Separator()

	b.Center("ANTI-NPC RULE")
	b.Text(rule)

	b.Separator()

	b.Center("STATUS: BOOTED")
	b.Center("PHONE: FORBIDDEN")
	b.Center("DISCIPLINE: LOADING")

	b.Separator()

	return b.String(), nil
}

// todo вынести эту куда-то в норм место + придумать генерацию
func pickWakeCommand(wakePlan storage.WakePlan) string {
	commands := []string{
		"Сегодня задача простая: поднять корпус, не открыть телефон и не начать день как NPC на автопилоте.",
		"Первый квест утра: встать. Второй: не залипнуть. Третий: дождаться чека с метриками и не развалиться морально.",
		"Организм загружается. Не мешай ему лентой, уведомлениями и прочим цифровым болотом.",
		"Встал — уже победа. Осталось не проиграть её телефону в первые три минуты.",
		"Сначала ноги на пол, потом реальность. Телефон пусть лежит, он сегодня не начальник смены.",
		"Утро не спрашивает, готов ли ты. Оно просто деплоит новый день в прод.",
		"Подъём. Без переговоров. Кровать уже проиграла по таймеру.",
		"Не надо геройства. Надо вертикальное положение и отсутствие телефона в руке.",
		"День стартовал. Не превращай boot sequence в doomscrolling sequence.",
		"Сейчас главное — не скорость, а факт запуска. Сервер поднялся, дальше разберёмся.",
	}

	return commands[deterministicIndex(wakePlan, "wake_command", len(commands))]
}

// todo вынести эту куда-то в норм место + придумать генерацию
func pickMorningRule(wakePlan storage.WakePlan) string {
	rules := []string{
		"Телефон утром трогают слабые. Сильные ждут чек.",
		"Кто открыл ленту до первого шага — тот сам себе баг в проде.",
		"Пацан может быть сонный, но не обязан быть алгоритмом рекомендаций.",
		"Сначала жизнь, потом уведомления. Не наоборот, чемпион.",
		"Палец тянется к телефону — это не воля, это кривой cron.",
		"Если день начинается с экрана, значит NPC mode ещё не выключен.",
		"Проснуться мало. Надо ещё не слить утро в цифровую канаву.",
		"Сон закончился. Самоуважение только начинается.",
		"Утренний прод поднят. Не ломай его TikTok-миграцией.",
		"Кровать отпустила. Телефон пусть ждёт в очереди.",
	}

	return rules[deterministicIndex(wakePlan, "morning_rule", len(rules))]
}

func deterministicIndex(wakePlan storage.WakePlan, salt string, size int) int {
	if size <= 1 {
		return 0
	}

	h := fnv.New64a()

	_, _ = h.Write([]byte(fmt.Sprintf(
		"user=%d|wake_plan=%d|date=%s|salt=%s",
		wakePlan.UserID,
		wakePlan.ID,
		wakePlan.Date.UTC().Format("2006-01-02"),
		salt,
	)))

	return int(h.Sum64() % uint64(size))
}
