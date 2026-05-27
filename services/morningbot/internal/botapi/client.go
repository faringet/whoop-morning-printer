package botapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	pcfg "github.com/faringet/whoop-morning-printer/pkg/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client struct {
	log         *slog.Logger
	bot         *tgbotapi.BotAPI
	pollTimeout int
}

func New(cfg pcfg.TelegramBot, log *slog.Logger) (*Client, error) {
	if log == nil {
		log = slog.Default()
	}

	if strings.TrimSpace(cfg.Token) == "" {
		return nil, errors.New("morningbot botapi: token is required")
	}

	log = log.With(
		slog.String("layer", "transport"),
		slog.String("module", "morningbot.botapi"),
	)

	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("morningbot botapi: init: %w", err)
	}

	bot.Debug = cfg.Debug

	timeout := int(cfg.PollTimeout.Seconds())
	if timeout <= 0 {
		timeout = 30
	}

	return &Client{
		log:         log,
		bot:         bot,
		pollTimeout: timeout,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.bot == nil {
		return errors.New("morningbot botapi: client is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	user, err := c.bot.GetMe()
	if err != nil {
		return fmt.Errorf("morningbot botapi: getMe: %w", err)
	}

	c.log.Info("botapi ready",
		slog.String("username", "@"+user.UserName),
		slog.Int64("id", user.ID),
		slog.Bool("is_bot", user.IsBot),
	)

	return nil
}

func (c *Client) SendHTML(ctx context.Context, chatID int64, text string, disablePreview bool) error {
	if c == nil || c.bot == nil {
		return errors.New("morningbot botapi: client is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if chatID == 0 {
		return errors.New("morningbot botapi: chat_id is required")
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("morningbot botapi: text is required")
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = disablePreview

	if _, err := c.bot.Send(msg); err != nil {
		return fmt.Errorf("morningbot botapi: send message: %w", err)
	}

	return nil
}

func (c *Client) Listen(ctx context.Context, handler func(context.Context, tgbotapi.Update) error) error {
	if c == nil || c.bot == nil {
		return errors.New("morningbot botapi: client is nil")
	}
	if handler == nil {
		return errors.New("morningbot botapi: handler is nil")
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = c.pollTimeout

	updates := c.bot.GetUpdatesChan(updateConfig)
	defer c.bot.StopReceivingUpdates()

	c.log.Info("telegram polling started",
		slog.Int("poll_timeout_seconds", c.pollTimeout),
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case update, ok := <-updates:
			if !ok {
				c.log.Info("telegram updates channel closed")
				return nil
			}

			if err := handler(ctx, update); err != nil {
				c.log.Warn("telegram update handler failed",
					slog.Int("update_id", update.UpdateID),
					slog.Any("err", err),
				)
			}
		}
	}
}
