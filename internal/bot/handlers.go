package tgbot

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"example.com/bot/internal/repository"
	"github.com/go-telegram/bot"
	m "github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

const (
	pattern = `^(?P<hours_10>\d+)(?P<hours_1>\d+)(?P<mins_10>\d+)(?P<mins_1>\d+)`
)

type TelegramBotHandlers struct {
	r            *repository.Dao
	storage      *repository.LocalStorage
	mes          sync.Map
	timpePattern *regexp.Regexp
	subexpNames  []string
}

func NewTgHandlers(r *repository.Dao, storage *repository.LocalStorage) *TelegramBotHandlers {
	regExp := regexp.MustCompile(pattern)
	subexpNames := regExp.SubexpNames()
	return &TelegramBotHandlers{
		r:            r,
		storage:      storage,
		mes:          sync.Map{},
		timpePattern: regExp,
		subexpNames:  subexpNames,
	}
}

func (th *TelegramBotHandlers) processTimeTracking(ctx context.Context, b *bot.Bot, update *m.Update) {
	chatID := update.Message.Chat.ID
	log := logger.Log.With(
		zap.String("handler", "defaultHandler"),
		zap.Int64("chatID", chatID),
	)

	if update.Message.Text == "/ignore_task" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "OK, no tracking for this task",
		})
		th.storage.SetStatus(chatID, repository.NoActionState)
		return
	}

	taskVal, ok := th.mes.Load(update.Message.ReplyToMessage.ID)
	if ok {
		task := taskVal.(models.WebHookParsed)
		matches := th.timpePattern.FindStringSubmatch(update.Message.Text)

		if matches == nil {
			log.Debug("Invalid time format received",
				zap.String("input", update.Message.Text))

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Incorrect time format. Please use format HHMM (e.g., 0130 for 1h 30m)",
			})
			return
		}

		result := make(map[string]uint32)
		for i, name := range th.subexpNames {
			if i > 0 && name != "" && i < len(matches) {
				dig, err := strconv.Atoi(matches[i])
				if err != nil {
					log.Error("Failed to parse time digit",
						zap.String("digit", matches[i]),
						zap.String("component", name),
						zap.Error(err))

					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: chatID,
						Text:   "Invalid time format. Please use format HHMM (e.g., 0130 for 1h 30m)",
					})
					return
				}
				result[name] = uint32(dig)
			}
		}

		timeSpent := uint32(0)
		for key, value := range result {
			switch key {
			case "hours_10":
				timeSpent += value * 600
			case "hours_1":
				timeSpent += value * 60
			case "mins_10":
				timeSpent += value * 10
			case "mins_1":
				timeSpent += value
			}
		}

		task.TimeSpent = timeSpent

		hours := timeSpent / 60
		mins := timeSpent % 60
		timeFormatted := ""
		if hours > 0 {
			timeFormatted = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			timeFormatted = fmt.Sprintf("%dm", mins)
		}

		if err := th.r.StoreTaskTracked(ctx, chatID, task); err != nil {
			log.Error("Failed to store task",
				zap.String("task", task.Task),
				zap.Uint32("timeSpent", task.TimeSpent),
				zap.Error(err))

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Failed to store task. Please try again later.",
			})
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("Task successfully tracked:\n• %s\n• Time spent: %s", task.Task, timeFormatted),
		})

		th.mes.Delete(update.Message.ReplyToMessage.ID)
		return
	}
}

func (th *TelegramBotHandlers) defaultHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	if update.Message == nil {
		logger.Log.Debug("Update received without message",
			zap.Any("update", update),
		)
		return
	}

	if update.Message.ReplyToMessage != nil {
		th.processTimeTracking(ctx, b, update)
		return
	}

	chatID := update.Message.Chat.ID

	state := th.storage.GetStatus(chatID)
	switch state {
	case repository.TodoistRegisteringState:
		if update.Message.Text == "/cancel" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Registration cancelled. You can complete it later with /auth.",
			})
			th.storage.SetStatus(chatID, repository.NoActionState)
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Please complete the registration process or use /cancel to stop.",
			})
		}
	case repository.WaitingForTimeToTrackState:
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Please reply to the task message to add time tracking.",
		})
	}
}

func (th *TelegramBotHandlers) startHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	if update.Message == nil {
		return
	}

	log := logger.Log.With(
		zap.String("handler", "startHandler"),
		zap.Int64("chatID", update.Message.Chat.ID),
	)

	u := &models.TgUser{
		ChatID: update.Message.Chat.ID,
		Name:   update.Message.Chat.Username,
	}

	isNewUser, err := th.r.CreateUser(ctx, u)
	if err != nil {
		log.Error("Failed to create/check user", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: u.ChatID,
			Text:   "Sorry, there was a problem registering you. Please try again later.",
		})
		return
	}

	if !isNewUser {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: u.ChatID,
			Text:   "Welcome back! Use /help to see available commands.",
		})
		return
	}

	welcomeMsg := "👋 *Welcome to the Time Tracker Bot!*\n\n" +
		"This bot helps you track time spent on your Todoist tasks.\n\n" +
		"To get started:\n" +
		"1. Connect your Todoist account using /auth\n" +
		"2. Complete tasks in Todoist with @track label or duration\n" +
		"3. View your stats with /stats\n\n" +
		"Type /help anytime to see available commands."

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    u.ChatID,
		Text:      welcomeMsg,
		ParseMode: m.ParseModeMarkdown,
	})
}

func (th *TelegramBotHandlers) helpHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "/auth\n/stats\n/help",
	})
}

func (th *TelegramBotHandlers) statsHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	log := logger.Log.With(
		zap.String("handler", "statsHandler"),
		zap.Int64("chatID", update.Message.Chat.ID),
	)

	chatID := update.Message.Chat.ID
	timeSpent, tasks, err := th.r.GetUserStats(ctx, chatID)
	if err != nil {
		log.Error("Failed to get user stats", zap.Error(err))
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Failed to retrieve your statistics. Please try again later.",
		})
		return
	}

	if len(tasks) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "You don't have any tracked tasks yet.",
		})
		return
	}

	res := fmt.Sprintf("📊 *Your Time Statistics*\n\nTotal time spent: %d minutes\n\n", timeSpent)
	res += "*Tasks:*\n"

	maxTasksToShow := 15
	displayCount := len(tasks)
	if displayCount > maxTasksToShow {
		displayCount = maxTasksToShow
	}

	for i := 0; i < displayCount; i++ {
		t := tasks[i]
		hours := t.TimeSpent / 60
		mins := t.TimeSpent % 60
		timeFormatted := ""

		if hours > 0 {
			timeFormatted = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			timeFormatted = fmt.Sprintf("%dm", mins)
		}

		res += fmt.Sprintf("• %s - %s\n", t.Task, timeFormatted)
	}

	if len(tasks) > maxTasksToShow {
		res += fmt.Sprintf("\n...and %d more tasks", len(tasks)-maxTasksToShow)
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      res,
		ParseMode: m.ParseModeMarkdown,
	})
}

func (th *TelegramBotHandlers) authHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	chatID := update.Message.Chat.ID
	ch := strconv.FormatInt(chatID, 10)
	link := "https://snbn.online/auth?chat_id=" + ch
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      "Auth using this [link](" + link + ")",
		ParseMode: m.ParseModeMarkdown,
	})
	th.storage.SetStatus(chatID, repository.TodoistRegisteringState)
}
