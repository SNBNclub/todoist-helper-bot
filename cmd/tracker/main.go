package main

import (
	"context"
	"os"
	"os/signal"
	"sync"

	// "sync"

	config "example.com/bot/configs"
	"example.com/bot/internal/logger"

	"go.uber.org/zap"

	_ "example.com/bot/internal/service/webhook"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	logger.Log.Debug("starting config creating")
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	logger.Log.Debug("fnish config creating",
		zap.Any("config", cfg),
	)
	logger.Log.Debug("Process",
		zap.Int("PID:", os.Getpid()),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		// bot.WithDefaultHandler(defaultHandler),
	}

	// token := os.Getenv("TELEGRAM_APITOKEN")
	token := "7585886902:AAFm79BfXyO7p328aFyMWRHfP6ojR7ZI9qI"

	b, err := bot.New(token, opts...)
	if nil != err {
		panic(err)
	}

	b.RegisterHandlerMatchFunc(matchFunc, handleAdd)
	// b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, startHandler)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		b.Start(ctx)
	}()

	wg.Wait()
}

func matchFunc(update *models.Update) bool {
	if update.Message == nil {
		return false
	}
	return update.Message.Text == "set_todoist_id"
}

var userStatuses = make(map[int64]int64)

const (
	waitingForID = iota
)

func handleAdd(ctx context.Context, b *bot.Bot, update *models.Update) {
	userStatuses[update.Message.Chat.ID] = waitingForID
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// b.SendMessage(ctx, &bot.SendMessageParams{
	// 	ChatID:    update.Message.Chat.ID,
	// 	Text:      "Hello, *" + bot.EscapeMarkdown(update.Message.From.FirstName) + "*",
	// 	ParseMode: models.ParseModeMarkdown,
	// })
	if update.Message == nil {
		return
	}
	if userStatuses[update.Message.Chat.ID] == waitingForID {
		err := dao.DAO.AddUserId(ctx, update.Message.Chat.ID, update.Message.Text)
		if err != nil {
			panic(err)
		}
	}

}

func startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	u := m.User{ChatID: update.Message.Chat.ID, Name: update.Message.Chat.Username}
	isNewUser, err := dao.DAO.CreateUser(ctx, &u)
	if err != nil {
		//TODO::try again
	}
	if !isNewUser {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "already reg",
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	kb := &models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{
				{Text: "set_todoist_id"},
				{Text: "Option 2"},
			},
			{
				{Text: "Option 3"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Please select an option from the menu below:",
		ReplyMarkup: kb,
	})
}
