package tgbot

import (
	"context"
	"fmt"
	"sync"

	"example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"github.com/go-telegram/bot"
)

type TelegramBotApi struct {
	b                 *bot.Bot
	h                 *TelegramBotHandlers
	authNotifications <-chan models.AuthNotification
	wh                <-chan models.WebHookParsed
	tq                map[int64]chan models.WebHookParsed
	tp                map[int64]map[string]models.WebHookParsed
}

func New(TelegramTokenAPI string, debugHandler bot.DebugHandler, handlers *TelegramBotHandlers, authNotificationsChan <-chan models.AuthNotification, webHookChan <-chan models.WebHookParsed) (*TelegramBotApi, error) {
	opts := []bot.Option{
		bot.WithDefaultHandler(handlers.defaultHandler),
		bot.WithDebugHandler(debugHandler),
		bot.WithWorkers(8),
	}
	b, err := bot.New(TelegramTokenAPI, opts...)
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handlers.startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, handlers.helpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stats", bot.MatchTypeExact, handlers.statsHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/auth", bot.MatchTypeExact, handlers.authHandler)

	return &TelegramBotApi{b: b,
		h:                 handlers,
		authNotifications: authNotificationsChan,
		wh:                webHookChan,
		tq:                make(map[int64]chan models.WebHookParsed),
		tp:                make(map[int64]map[string]models.WebHookParsed),
	}, nil
}

func (b *TelegramBotApi) Start(wg *sync.WaitGroup, ctx context.Context) {
	wg.Add(1)

	go func() {
		defer wg.Done()

		b.b.Start(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case notification := <-b.authNotifications:
				if notification.Successful {
					b.b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: notification.ChatID,
						Text:   "Todoist authentication completed successfully!",
					})
				} else {
					b.b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: notification.ChatID,
						Text:   "Todoist authentication failed. Please try again.",
					})
					b.h.storage.SetStatus(notification.ChatID, noActionState)
				}
			}
		}
	}()

	go b.AskToTrackTime(wg, ctx)
}

func (b *TelegramBotApi) AskToTrackTime(wg *sync.WaitGroup, ctx context.Context) {
	logger.Log.Debug("run ask to track time")
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case val := <-b.wh:
				logger.Log.Debug("get webhook")
				chatID := b.h.r.GetChatIDByTodoist(ctx, val.UserID)
				if !val.AskTime {
					b.b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: chatID,
						Text:   fmt.Sprintf("Stored %d for task: %s", val.TimeSpent, val.Task),
					})
					b.h.r.StoreTaskTracked(ctx, chatID, val)
					continue
				}
				msg, _ := b.b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   fmt.Sprintf("Enter time for this task in reply message: %s", val.Task),
				})
				b.h.mes.Store(msg.ID, val)
			}
		}
	}()
}
