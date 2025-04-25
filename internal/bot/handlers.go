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
	pattern       = `^(?P<hours_10>\d+)(?P<hours_1>\d+)(?P<mins_10>\d+)(?P<mins_1>\d+)`
	noActionState = iota
	todoistRegisteringState
	waitingForTimeToTrackState
)

type TelegramBotHandlers struct {
	// r       DaoInterface
	r       *repository.Dao
	storage *repository.LocalStorage
	mes     sync.Map
}

// type DaoInterface interface {
// 	CreateUser(ctx context.Context, user *models.TgUser) (bool, error)
// 	AddTodoistUser(ctx context.Context, todoistID, userName string) error
// 	AddUserId(ctx context.Context, chatID int64, todoistID string) error
// 	Close()
// }

func NewTgHandlers(r *repository.Dao, storage *repository.LocalStorage) *TelegramBotHandlers {
	return &TelegramBotHandlers{
		r:       r,
		storage: storage,
	}
}

func (th *TelegramBotHandlers) defaultHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	if update.Message == nil {
		logger.Log.Debug("update received in default",
			zap.Any("update", update),
		)
		return
	}
	chatID := update.Message.Chat.ID
	if update.Message.ReplyToMessage != nil {
		if update.Message.Text == "/ignore_task" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "OK, no tracking for this task",
			})
			return
		}
		val, ok := th.mes.Load(update.Message.ReplyToMessage.ID)
		if ok {
			val := val.(models.WebHookParsed)
			r := regexp.MustCompile(pattern)
			matches := r.FindStringSubmatch(update.Message.Text)
			if matches == nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   "Uncorrect time format",
				})
				return
			}
			result := make(map[string]uint32)
			for i, name := range r.SubexpNames() {
				if i > 0 && name != "" && i < len(matches) {
					dig, err := strconv.Atoi(matches[i])
					if err != nil {
						panic(err)
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
			val.TimeSpent = timeSpent
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   fmt.Sprintf("Task: %s succesfully tracked: %d", val.Task, val.TimeSpent),
			})
			th.r.StoreTaskTracked(ctx, chatID, val)
			th.mes.Delete(update.Message.ReplyToMessage.ID)
		}
	}
	state := th.storage.GetStatus(chatID)
	switch state {
	case todoistRegisteringState:
		if update.Message.Text == "/cancel" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Finish reg later",
			})
			th.storage.SetStatus(chatID, noActionState)
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Use /cancel or finish the reg",
			})
		}
	}
}

func (th *TelegramBotHandlers) startHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	if update.Message == nil {
		return
	}
	u := &models.TgUser{
		ChatID: update.Message.Chat.ID,
		Name:   update.Message.Chat.Username,
	}
	// TODO :: fix true/false for exists
	isNewUser, err := th.r.CreateUser(ctx, u)
	// exist = !exist
	if err != nil {
		panic(err)
	}
	if !isNewUser {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: u.ChatID,
			Text:   "already reg!",
		})
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: u.ChatID,
		Text:   "Hello!",
	})
}

func (th *TelegramBotHandlers) helpHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "/auth\n/stats/help",
	})
}

func (th *TelegramBotHandlers) statsHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	chatID := update.Message.Chat.ID
	timeSpent, tasks := th.r.GetUserStats(ctx, chatID)
	res := fmt.Sprintf("You spent: %d\n", timeSpent)
	for _, t := range tasks {
		res += fmt.Sprintf("Task: %s - %d\n", t.Task, t.TimeSpent)
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   res,
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
	th.storage.SetStatus(chatID, todoistRegisteringState)
}
