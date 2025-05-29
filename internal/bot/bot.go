package tgbot

import (
	"context"
	"sync"

	"github.com/go-telegram/bot"
)

type TelegramBotApi struct {
	b       *bot.Bot
	service *TgBotService
	// h                            *TgBotHandlers
	// logger                       *zap.Logger
}

func New(TelegramTokenAPI string, debugHandler bot.DebugHandler, service *TgBotService, handlers *TgBotHandlers) (*TelegramBotApi, error) {
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

	return &TelegramBotApi{
		b:       b,
		service: service,
		// h: handlers,
	}, nil
}

func (b *TelegramBotApi) Start(wg *sync.WaitGroup, ctx context.Context) {
	wg.Add(1)

	// TODO :: bot can be stopped before
	go func() {
		defer wg.Done()

		b.b.Start(ctx)
	}()

	wg.Add(2)
	go func() {
		defer wg.Done()

		b.service.listenAuthNotifications(ctx, b.b)
	}()
	go func() {
		defer wg.Done()

		b.service.listenWebhooks(ctx, b.b)
	}()
}
