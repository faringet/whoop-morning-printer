package bottext

import (
	"fmt"
	"html"
	"strings"
	"time"
)

const (
	UnknownCommand = "🤔 Команда неизвестна. Сервер понял только одно: кто-то снова деплоит хаос.\n\nЖми /help — там список легальных заклинаний."
	//todo позднее дописать EmptyWakeTime
	//EmptyWakeTime  = "⏰ Время где, боец?\n\nПиши нормально: <code>/wake 08:30</code>\nЯ не экстрасенс, я просто бот с доступом к PostgreSQL."
	WakeTimeError = "⚠️ Это не время, это SQL-инъекция в здравый смысл.\n\nФормат такой: <code>HH:MM</code>\nНапример: <code>/wake 08:30</code>"
	InternalError = "⚠️ Что-то пошло не так. Я записал это в лог, потому что боль надо структурировать.\n\nПопробуй ещё раз. Если опять упадёт - значит утро уже началось."
)

func Start() string {
	return strings.Join([]string{
		"👋 Здарова. Я morningbot.",
		"",
		"Моя работа простая, как docker compose up:",
		"ты говоришь, во сколько тебя будить, а я создаю утренний план, чтобы потом чек с WHOOP-метриками вылез из принтера.",
		"",
		"<b>Главная команда:</b>",
		"• <code>/wake 08:30</code> — назначить пробуждение",
		"",
		"<b>Ещё есть:</b>",
		"• <code>/status</code> — проверить, поставлен ли утренний замес",
		"• <code>/cancel</code> — отменить план, если жизнь опять победила дисциплину",
		"• <code>/id</code> — показать твой Telegram user_id и chat_id",
		"• <code>/help</code> — открыть мануал для выживших",
		"",
		"Запомни: телефон утром трогают слабые. Сильные ждут чек.",
	}, "\n")
}

func Help(defaultWakeTime string, prepareBefore time.Duration, finalDeadlineAfter time.Duration) string {
	return strings.Join([]string{
		"🆘 <b>Помощь</b>",
		"",
		"<b>Команды, которые не стыдно знать:</b>",
		"• <code>/wake 08:30</code> — назначить пробуждение",
		"• <code>/wake tomorrow 08:30</code> — поставить будильник на завтра, без гаданий на кофейной гуще",
		"• <code>/status</code> — узнать, жив ли план или опять всё на честном слове",
		"• <code>/cancel</code> — отменить ближайший wake plan",
		"• <code>/id</code> — показать Telegram user_id и chat_id",
		"• <code>/help</code> — эта простыня, но полезная",
		"",
		"<b>Как работает эта шайтан-машина:</b>",
		"Ты пишешь <code>/wake 08:30</code>.",
		"Я создаю wake_plan и print_job.",
		"Потом другие сервисы подтянут WHOOP, сварят совет дня и отправят чек на печать.",
		"",
		fmt.Sprintf("⏰ Время по умолчанию: <code>%s</code>", html.EscapeString(defaultWakeTime)),
		fmt.Sprintf("🛠 Подготовка начинается за: <code>%s</code>", prepareBefore.String()),
		fmt.Sprintf("🏁 Финальный дедлайн отчёта: <code>%s</code> после пробуждения", finalDeadlineAfter.String()),
		"",
	}, "\n")
}

func ID(userID int64, chatID int64, username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		username = "не указан"
	} else {
		username = "@" + strings.TrimPrefix(username, "@")
	}

	return strings.Join([]string{
		"🪪 <b>Твои Telegram-реквизиты</b>",
		"",
		fmt.Sprintf("user_id: <code>%d</code>", userID),
		fmt.Sprintf("chat_id: <code>%d</code>", chatID),
		fmt.Sprintf("username: <code>%s</code>", html.EscapeString(username)),
		"",
		"Вот этот user_id добавляй в <code>access.allowed_user_ids</code>.",
		"Без него бот будет смотреть на тебя как Postgres на битый DSN.",
	}, "\n")
}

func WakePlanned(wakeAt time.Time, prepareAt time.Time, finalDeadlineAt time.Time, timezone string) string {
	return strings.Join([]string{
		"✅ <b>План пробуждения сохранён</b>",
		"",
		fmt.Sprintf("⏰ Подъём: <b>%s</b>", formatDateTime(wakeAt, timezone)),
		fmt.Sprintf("🛠 Подготовка системы: <code>%s</code>", formatDateTime(prepareAt, timezone)),
		fmt.Sprintf("🏁 Дедлайн финального отчёта: <code>%s</code>", formatDateTime(finalDeadlineAt, timezone)),
		"",
		"Wake plan создан. Print job тоже будет.",
		"Осталось только проснуться и не полезть в телефон как NPC.",
	}, "\n")
}

func StatusEmpty() string {
	return strings.Join([]string{
		"😴 Активного плана пробуждения нет.",
		"",
		"Система стоит без задачи, как джун без тикета.",
		"",
		"Назначь подъём:",
		"<code>/wake 08:30</code>",
	}, "\n")
}

func Status(wakeAt time.Time, prepareAt time.Time, finalDeadlineAt time.Time, status string, source string, timezone string) string {
	return strings.Join([]string{
		"📋 <b>Ближайший wake plan</b>",
		"",
		fmt.Sprintf("Статус: <b>%s</b>", html.EscapeString(status)),
		fmt.Sprintf("Источник: <code>%s</code>", html.EscapeString(source)),
		"",
		fmt.Sprintf("⏰ Подъём: <b>%s</b>", formatDateTime(wakeAt, timezone)),
		fmt.Sprintf("🛠 Подготовка: <code>%s</code>", formatDateTime(prepareAt, timezone)),
		fmt.Sprintf("🏁 Финальный отчёт до: <code>%s</code>", formatDateTime(finalDeadlineAt, timezone)),
		"",
		"План есть. Утро будет. Сопротивление бесполезно.",
	}, "\n")
}

func Cancelled(wakeAt time.Time, timezone string) string {
	return strings.Join([]string{
		"🛑 <b>Wake plan отменён</b>",
		"",
		fmt.Sprintf("Отменённый подъём: <code>%s</code>", formatDateTime(wakeAt, timezone)),
		"",
		"План снят с продакшена. Можно снова жить как человек, но недолго.",
	}, "\n")
}

func NothingToCancel() string {
	return "😶 Отменять нечего. Активного wake plan нет. Даже хаос сегодня по расписанию не пришёл."
}

func AccessDenied() string {
	return "⛔ Доступ закрыт. Этот бот не проходной двор, а элитный гараж утренней дисциплины."
}

func formatDateTime(t time.Time, timezone string) string {
	loc := time.UTC

	timezone = strings.TrimSpace(timezone)
	if timezone != "" {
		if loaded, err := time.LoadLocation(timezone); err == nil {
			loc = loaded
		}
	}

	return t.In(loc).Format("02.01.2006 15:04")
}
