package tgbot

import (
	"example.com/bot/internal/repository"
	"github.com/go-telegram/bot"
)

type TelegramBotApi struct {
	b *bot.Bot
	r *repository.Dao
	//
}

func New(TelegramTokenAPI string, debugHandler bot.DebugHandler) (*TelegramBotApi, error) {
	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithDebugHandler(debugHandler),
		bot.WithWorkers(8),
	}
	b, err := bot.New(TelegramTokenAPI, opts...)
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, helpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stats", bot.MatchTypeExact, startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/auth", bot.MatchTypeExact, authHandler)

	return &TelegramBotApi{b: b}, nil
}
