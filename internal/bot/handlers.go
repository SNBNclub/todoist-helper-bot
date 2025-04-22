package tgbot

import (
	"context"
	"regexp"
	"strconv"

	"example.com/bot/internal/models"
	"github.com/go-telegram/bot"
	m "github.com/go-telegram/bot/models"
)

const (
	pattern       = `^(?P<hours_10>\d+)(?P<hours_1>\d+):(?P<mins_10>\d+)(?P<mins_1>\d+)`
	noActionState = iota
	todoistRegisteringState
	TodoistRegfinishState
	waitingForTimeToTrackState
)

func (th *TelegramBotHandlers) defaultHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	chatID := update.Message.Chat.ID
	state := th.storage.GetStatus(chatID)
	switch state {
	case waitingForTimeToTrackState:
		if update.Message.Text == "/ignore_task" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "OK, no tracking for this task",
			})
			th.storage.SetStatus(chatID, noActionState)
		} else {
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
			for key, val := range result {
				switch key {
				case "hours_10":
					timeSpent += val * 600
				case "hours_1":
					timeSpent += val * 60
				case "mins_10":
					timeSpent += val * 10
				case "mins_1":
					timeSpent += val
				}
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Task succesfully tracked",
			})
			th.storage.SetStatus(chatID, noActionState)
			// write to queue to make send next task
		}
	case todoistRegisteringState:

		if update.Message.Text == "/cancel" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Finish reg later",
			})
			th.storage.SetStatus(chatID, noActionState)
		} else if update.Message.Text == "/regfinish" {
			// TODO :: deeplinks doesn't work.
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Congrats, you finish registration part",
			})
			th.storage.SetStatus(chatID, noActionState)
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Use /cancel or finish the reg",
			})
		}
	case TodoistRegfinishState:
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Congrats, you finish registration part",
		})
	}
}

func (th *TelegramBotHandlers) startHandler(ctx context.Context, b *bot.Bot, update *m.Update) {
	u := &models.TgUser{
		ChatID: update.Message.Chat.ID,
		Name:   update.Message.Chat.Username,
	}
	exist, err := th.r.CreateUser(ctx, u)
	if err != nil {
		panic(err)
	}
	if exist {
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
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "become later",
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
