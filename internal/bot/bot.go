package tgbot

import (
	"context"
	"fmt"
	"sync"

	"example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"example.com/bot/internal/repository"
	"github.com/go-telegram/bot"
	"go.uber.org/zap"
)

// AuthHandlerType defines the type of auth notification
type AuthHandlerType int

const (
	// AuthSuccess indicates successful authentication
	AuthSuccess AuthHandlerType = iota
	// AuthTimeout indicates authentication timed out
	AuthTimeout
	// AuthError indicates an error during authentication
	AuthError
)

// TelegramBotApi handles Telegram bot operations
type TelegramBotApi struct {
	b                 *bot.Bot
	h                 *TelegramBotHandlers
	authNotifications <-chan models.AuthNotification
	wh                <-chan models.WebHookParsed
	tq                map[int64]chan models.WebHookParsed
	tp                map[int64]map[string]models.WebHookParsed
}

// New creates a new Telegram bot instance
func New(telegramTokenAPI string, debugHandler bot.DebugHandler, handlers *TelegramBotHandlers, authNotificationsChan <-chan models.AuthNotification, webHookChan <-chan models.WebHookParsed) (*TelegramBotApi, error) {
	opts := []bot.Option{
		bot.WithDefaultHandler(handlers.defaultHandler),
		bot.WithDebugHandler(debugHandler),
		bot.WithWorkers(8),
	}

	b, err := bot.New(telegramTokenAPI, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}

	// Register command handlers
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handlers.startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, handlers.helpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stats", bot.MatchTypeExact, handlers.statsHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/auth", bot.MatchTypeExact, handlers.authHandler)

	return &TelegramBotApi{
		b:                 b,
		h:                 handlers,
		authNotifications: authNotificationsChan,
		wh:                webHookChan,
		tq:                make(map[int64]chan models.WebHookParsed),
		tp:                make(map[int64]map[string]models.WebHookParsed),
	}, nil
}

// Start begins the bot operations and notification handling
func (b *TelegramBotApi) Start(wg *sync.WaitGroup, ctx context.Context) {
	// Start Telegram bot
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Log.Info("Starting Telegram bot")
		b.b.Start(ctx)
		logger.Log.Info("Telegram bot stopped")
	}()

	// Handle auth notifications
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				logger.Log.Info("Auth notification handler stopping due to context cancellation")
				return

			case notification := <-b.authNotifications:
				log := logger.Log.With(
					zap.Int64("chatID", notification.ChatID),
					zap.Bool("successful", notification.Successful),
				)

				if notification.Error != nil {
					log = log.With(zap.Error(notification.Error))
				}

				if notification.Successful {
					log.Info("Todoist authentication successful")
					b.b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: notification.ChatID,
						Text:   "✅ Todoist authentication completed successfully! You can now track your tasks.",
					})
				} else {
					var message string

					// Handle different error types
					switch notification.Type {
					case int(AuthTimeout):
						log.Warn("Todoist authentication timeout")
						message = "⏰ Authentication timed out. Please try again with /auth."

					case int(AuthError):
						log.Error("Todoist authentication error")
						message = "❌ Authentication failed. Please try again later."

					default:
						log.Error("Todoist authentication failed with unknown reason")
						message = "❌ Authentication failed. Please try again with /auth."
					}

					b.b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: notification.ChatID,
						Text:   message,
					})

					// Reset user state
					b.h.storage.SetStatus(notification.ChatID, repository.NoActionState)
				}
			}
		}
	}()

	// Start webhook handler
	// FIXME :: never starts the go funciton like this
	go b.AskToTrackTime(wg, ctx)
}

// AskToTrackTime handles webhook events for task tracking
func (b *TelegramBotApi) AskToTrackTime(wg *sync.WaitGroup, ctx context.Context) {
	logger.Log.Info("Starting webhook task tracking handler")
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				logger.Log.Info("Task tracking handler stopping due to context cancellation")
				return

			case webhook := <-b.wh:
				log := logger.Log.With(
					zap.String("todoistUserID", webhook.UserID),
					zap.String("task", webhook.Task),
					zap.Uint32("timeSpent", webhook.TimeSpent),
					zap.Bool("askTime", webhook.AskTime),
				)
				log.Debug("Received webhook task tracking event")

				// Get chat ID from Todoist ID
				chatID, err := b.h.r.GetChatIDByTodoist(ctx, webhook.UserID)
				if err != nil {
					log.Error("Failed to get chat ID for Todoist user", zap.Error(err))
					continue
				}

				if chatID == 0 {
					log.Warn("No chat ID found for Todoist user")
					continue
				}

				// Handle automatic time tracking
				if !webhook.AskTime {
					// Calculate readable time format
					hours := webhook.TimeSpent / 60
					mins := webhook.TimeSpent % 60
					timeFormatted := ""
					if hours > 0 {
						timeFormatted = fmt.Sprintf("%dh %dm", hours, mins)
					} else {
						timeFormatted = fmt.Sprintf("%dm", mins)
					}

					// Store task in database
					if err := b.h.r.StoreTaskTracked(ctx, chatID, webhook); err != nil {
						log.Error("Failed to store tracked task", zap.Error(err))
						b.b.SendMessage(ctx, &bot.SendMessageParams{
							ChatID: chatID,
							Text:   "Failed to store task. Please try again later.",
						})
						continue
					}

					// Send confirmation to user
					b.b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: chatID,
						Text:   fmt.Sprintf("✅ Task tracked automatically:\n• %s\n• Time spent: %s", webhook.Task, timeFormatted),
					})
					continue
				}

				// Handle manual time tracking request
				log.Debug("Asking user for time tracking input")
				msg, err := b.b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    chatID,
					Text:      fmt.Sprintf("⏱️ Enter time for this task in HHMM format (e.g., 0130 for 1h 30m):\n\n*%s*", webhook.Task),
					ParseMode: "Markdown",
				})

				if err != nil {
					log.Error("Failed to send time tracking prompt", zap.Error(err))
					continue
				}

				// Store the message ID with the webhook data for later retrieval
				b.h.mes.Store(msg.ID, webhook)
			}
		}
	}()
}
