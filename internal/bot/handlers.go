package tgbot

import (
	"context"

	"github.com/go-telegram/bot"
	m "github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

type TgBotHandlers struct {
	service *TgBotService
	logger  *zap.Logger
}

func NewTgBotHandlers(service *TgBotService, logger *zap.Logger) *TgBotHandlers {
	return &TgBotHandlers{
		service: service,
		logger:  logger,
	}
}

func (th *TgBotHandlers) defaultHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	logger := th.logger.With(
		zap.String("ThBotHandlers", "defaultHandler"),
	)
	sendMessageParams := th.service.processUpdate(ctx, update)
	if sendMessageParams != nil {
		_, err := b.SendMessage(ctx, sendMessageParams)
		if err != nil {
			logger.Error("error trying to send message",
				zap.Error(err),
			)
		}
	}
}

func (th *TgBotHandlers) startHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	logger := th.logger.With(
		zap.String("TgBotHandlers", "startHandler"),
	)
	sendMessageParams := th.service.processStart(ctx, update)
	if sendMessageParams != nil {
		_, err := b.SendMessage(ctx, sendMessageParams)
		if err != nil {
			logger.Error("error trying to send message",
				zap.Error(err),
			)
		}
	}
}

func (th *TgBotHandlers) helpHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "/auth\n/stats\n/help",
	})
}

func (th *TgBotHandlers) statsHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	logger := th.logger.With(
		zap.String("TgBotHandlers", "statsHandler"),
	)
	sendMessageParams := th.service.processStats(ctx, update)
	if sendMessageParams != nil {
		_, err := b.SendMessage(ctx, sendMessageParams)
		if err != nil {
			logger.Error("error trying to send message",
				zap.Error(err),
			)
		}
	}
}

func (th *TgBotHandlers) authHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	logger := th.logger.With(
		zap.String("TgBotHandlers", "authHandler"),
	)
	sendMessageParams := th.service.processAuth(update)
	if sendMessageParams != nil {
		_, err := b.SendMessage(ctx, sendMessageParams)
		if err != nil {
			logger.Error("error trying to send message",
				zap.Error(err),
			)
		}
	}
}
