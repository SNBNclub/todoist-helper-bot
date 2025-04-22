package tgbot

import (
	"context"
	"sync"

	"example.com/bot/internal/repository"
	"github.com/go-telegram/bot"
)

type TelegramBotApi struct {
	b *bot.Bot
	//
}

type TelegramBotHandlers struct {
	r       *repository.Dao
	storage *repository.LocalStorage
}

func NewTgHandlers(r *repository.Dao, storage *repository.LocalStorage) *TelegramBotHandlers {
	return &TelegramBotHandlers{
		r:       r,
		storage: storage,
	}
}

func New(TelegramTokenAPI string, debugHandler bot.DebugHandler, handlers *TelegramBotHandlers) (*TelegramBotApi, error) {
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

	return &TelegramBotApi{b: b}, nil
}

func (b *TelegramBotApi) Start(wg *sync.WaitGroup, ctx context.Context) {
	wg.Add(1)

	go func() {
		defer wg.Done()

		b.b.Start(ctx)
	}()
}
