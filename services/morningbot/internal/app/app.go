package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	platformpg "github.com/faringet/whoop-morning-printer/internal/platform/postgres"
	"github.com/faringet/whoop-morning-printer/services/morningbot/config"
	"github.com/faringet/whoop-morning-printer/services/morningbot/internal/botapi"
	"github.com/faringet/whoop-morning-printer/services/morningbot/internal/bottext"
	"github.com/faringet/whoop-morning-printer/services/morningbot/internal/orchestrator"
	"github.com/faringet/whoop-morning-printer/services/morningbot/internal/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type App struct {
	cfg *config.Config
	log *slog.Logger

	store        storage.Store
	bot          *botapi.Client
	orchestrator *orchestrator.Orchestrator
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	if cfg == nil {
		return nil, errors.New("morningbot app: config is nil")
	}
	if log == nil {
		return nil, errors.New("morningbot app: logger is nil")
	}

	rootLog := log
	appLog := log.With(
		slog.String("layer", "app"),
		slog.String("module", "morningbot.app"),
	)

	st, err := openStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	bot, err := botapi.New(cfg.TelegramBot, rootLog)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create bot client: %w", err)
	}

	orch, err := orchestrator.New(st, orchestrator.Config{
		UserID: cfg.MorningBot.UserID,

		Timezone:        cfg.MorningBot.Timezone,
		DefaultWakeTime: cfg.MorningBot.DefaultWakeTime,

		PrepareBefore:      cfg.MorningBot.PrepareBefore,
		FinalDeadlineAfter: cfg.MorningBot.FinalDeadlineAfter,
	})
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("create orchestrator: %w", err)
	}

	return &App{
		cfg:          cfg,
		log:          appLog,
		store:        st,
		bot:          bot,
		orchestrator: orch,
	}, nil
}

func openStore(cfg *config.Config) (storage.Store, error) {
	if cfg == nil {
		return nil, errors.New("morningbot app: config is nil")
	}

	db, err := platformpg.Open(platformpg.Config{
		DSN:             cfg.Storage.Postgres.DSN,
		MaxOpenConns:    cfg.Storage.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Storage.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Storage.Postgres.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Storage.Postgres.ConnMaxIdleTime,
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres db: %w", err)
	}

	st, err := storage.NewPostgres(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create postgres storage: %w", err)
	}

	return st, nil
}

func (a *App) Close() error {
	if a == nil || a.store == nil {
		return nil
	}

	return a.store.Close()
}

func (a *App) Run(ctx context.Context) error {
	if a == nil {
		return errors.New("morningbot app: app is nil")
	}

	a.log.Info("run started",
		slog.Int64("user_id", a.cfg.MorningBot.UserID),
		slog.String("timezone", a.cfg.MorningBot.Timezone),
		slog.String("default_wake_time", a.cfg.MorningBot.DefaultWakeTime),
		slog.Duration("prepare_before", a.cfg.MorningBot.PrepareBefore),
		slog.Duration("final_deadline_after", a.cfg.MorningBot.FinalDeadlineAfter),
		slog.Int("allowed_user_ids_count", len(a.cfg.Access.AllowedUserIDs)),
		slog.Int("allowed_chat_ids_count", len(a.cfg.Access.AllowedChatIDs)),
	)

	if len(a.cfg.Access.AllowedUserIDs) == 0 && len(a.cfg.Access.AllowedChatIDs) == 0 {
		a.log.Warn("morningbot access is open to everyone")
	}

	if err := a.bot.Ping(ctx); err != nil {
		return err
	}

	return a.bot.Listen(ctx, a.handleUpdate)
}

func (a *App) handleUpdate(ctx context.Context, update tgbotapi.Update) error {
	if update.Message == nil {
		return nil
	}

	msg := update.Message

	var fromID int64
	if msg.From != nil {
		fromID = msg.From.ID
	}

	if !a.isAllowed(fromID, msg.Chat.ID) {
		a.log.Warn("access denied",
			slog.Int64("from_id", fromID),
			slog.Int64("chat_id", msg.Chat.ID),
			slog.String("username", safeUsername(msg)),
		)

		denyMessage := strings.TrimSpace(a.cfg.Access.DenyMessage)
		if denyMessage == "" {
			denyMessage = bottext.AccessDenied()
		}

		if msg.IsCommand() {
			return a.bot.SendHTML(ctx, msg.Chat.ID, denyMessage, true)
		}

		return nil
	}

	if !msg.IsCommand() {
		text := strings.TrimSpace(msg.Text)
		if text == "" {
			return nil
		}

		return a.bot.SendHTML(
			ctx,
			msg.Chat.ID,
			"🤖 Я пока понимаю только команды. Не надо со мной как с человеком, я всего лишь if/switch с завышенной самооценкой.\n\nЖми <code>/help</code>.",
			true,
		)
	}

	switch msg.Command() {
	case "start":
		return a.replyStart(ctx, msg.Chat.ID)

	case "help":
		return a.replyHelp(ctx, msg.Chat.ID)

	case "id":
		return a.replyID(ctx, msg)

	case "wake":
		return a.replyWake(ctx, msg)

	case "status":
		return a.replyStatus(ctx, msg.Chat.ID)

	case "cancel":
		return a.replyCancel(ctx, msg.Chat.ID)

	case "testprint":
		return a.replyTestPrint(ctx, msg.Chat.ID)

	default:
		return a.bot.SendHTML(ctx, msg.Chat.ID, bottext.UnknownCommand, true)
	}
}

func (a *App) replyStart(ctx context.Context, chatID int64) error {
	return a.bot.SendHTML(ctx, chatID, bottext.Start(), true)
}

func (a *App) replyHelp(ctx context.Context, chatID int64) error {
	return a.bot.SendHTML(
		ctx,
		chatID,
		bottext.Help(
			a.cfg.MorningBot.DefaultWakeTime,
			a.cfg.MorningBot.PrepareBefore,
			a.cfg.MorningBot.FinalDeadlineAfter,
		),
		true,
	)
}

func (a *App) replyID(ctx context.Context, msg *tgbotapi.Message) error {
	if msg == nil {
		return nil
	}

	var userID int64
	var username string

	if msg.From != nil {
		userID = msg.From.ID
		username = msg.From.UserName
	}

	return a.bot.SendHTML(ctx, msg.Chat.ID, bottext.ID(userID, msg.Chat.ID, username), true)
}

func (a *App) replyWake(ctx context.Context, msg *tgbotapi.Message) error {
	if msg == nil {
		return nil
	}

	command, err := orchestrator.ParseWakeArgs(msg.CommandArguments(), a.cfg.MorningBot.DefaultWakeTime)
	if err != nil {
		a.log.Warn("parse wake command failed",
			slog.Int64("chat_id", msg.Chat.ID),
			slog.String("args", msg.CommandArguments()),
			slog.Any("err", err),
		)

		return a.bot.SendHTML(ctx, msg.Chat.ID, bottext.WakeTimeError, true)
	}

	var telegramUserID *int64
	if msg.From != nil && msg.From.ID > 0 {
		id := msg.From.ID
		telegramUserID = &id
	}

	result, err := a.orchestrator.ScheduleWake(ctx, orchestrator.ScheduleWakeInput{
		Command:        command,
		TelegramUserID: telegramUserID,
	})
	if err != nil {
		if errors.Is(err, orchestrator.ErrWakeTimeInPast) {
			return a.bot.SendHTML(
				ctx,
				msg.Chat.ID,
				"⚠️ Сегодня это время уже прошло. Машину времени не завезли.\n\nПиши так: <code>/wake tomorrow 08:30</code>",
				true,
			)
		}

		a.log.Error("schedule wake failed",
			slog.Int64("chat_id", msg.Chat.ID),
			slog.String("args", msg.CommandArguments()),
			slog.Any("err", err),
		)

		return a.bot.SendHTML(ctx, msg.Chat.ID, bottext.InternalError, true)
	}

	return a.bot.SendHTML(
		ctx,
		msg.Chat.ID,
		bottext.WakePlanned(
			result.WakePlan.WakeAt,
			result.WakePlan.PrepareAt,
			result.WakePlan.FinalDeadlineAt,
			a.cfg.MorningBot.Timezone,
		),
		true,
	)
}

func (a *App) replyStatus(ctx context.Context, chatID int64) error {
	result, err := a.orchestrator.Status(ctx)
	if errors.Is(err, storage.ErrNotFound) {
		return a.bot.SendHTML(ctx, chatID, bottext.StatusEmpty(), true)
	}
	if err != nil {
		a.log.Error("get wake status failed", slog.Any("err", err))
		return a.bot.SendHTML(ctx, chatID, bottext.InternalError, true)
	}

	return a.bot.SendHTML(
		ctx,
		chatID,
		bottext.Status(
			result.WakePlan.WakeAt,
			result.WakePlan.PrepareAt,
			result.WakePlan.FinalDeadlineAt,
			string(result.WakePlan.Status),
			string(result.WakePlan.Source),
			a.cfg.MorningBot.Timezone,
		),
		true,
	)
}

func (a *App) replyCancel(ctx context.Context, chatID int64) error {
	result, err := a.orchestrator.Cancel(ctx)
	if errors.Is(err, storage.ErrNotFound) {
		return a.bot.SendHTML(ctx, chatID, bottext.NothingToCancel(), true)
	}
	if err != nil {
		a.log.Error("cancel wake plan failed", slog.Any("err", err))
		return a.bot.SendHTML(ctx, chatID, bottext.InternalError, true)
	}

	return a.bot.SendHTML(
		ctx,
		chatID,
		bottext.Cancelled(result.WakePlan.WakeAt, a.cfg.MorningBot.Timezone),
		true,
	)
}

func (a *App) replyTestPrint(ctx context.Context, chatID int64) error {
	result, err := a.orchestrator.CreateTestPrintJob(ctx)
	if err != nil {
		a.log.Error("create test print job failed", slog.Any("err", err))
		return a.bot.SendHTML(ctx, chatID, bottext.InternalError, true)
	}

	text := strings.Join([]string{
		"🧾 <b>Тестовая задача печати создана</b>",
		"",
		fmt.Sprintf("print_job_id: <code>%d</code>", result.PrintJob.ID),
		fmt.Sprintf("status: <code>%s</code>", result.PrintJob.Status),
		fmt.Sprintf("type: <code>%s</code>", result.PrintJob.Type),
		"",
		"Очередь печати получила маленький пинок. Теперь printeragent в будущем должен это подобрать.",
	}, "\n")

	return a.bot.SendHTML(ctx, chatID, text, true)
}

func safeUsername(msg *tgbotapi.Message) string {
	if msg == nil || msg.From == nil || strings.TrimSpace(msg.From.UserName) == "" {
		return ""
	}

	return "@" + msg.From.UserName
}
