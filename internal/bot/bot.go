package tgbot

import (
	"context"
	"fmt"
	"sync"

	"example.com/bot/internal/models"
	"github.com/go-telegram/bot"
)

type TelegramBotApi struct {
	b  *bot.Bot
	h  *TelegramBotHandlers
	wh <-chan models.WebHookParsed
	tq map[int64]chan models.WebHookParsed
	tp map[int64]map[string]models.WebHookParsed
}

func New(TelegramTokenAPI string, debugHandler bot.DebugHandler, handlers *TelegramBotHandlers, webHookChan <-chan models.WebHookParsed) (*TelegramBotApi, error) {
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

	return &TelegramBotApi{b: b, wh: webHookChan, tq: make(map[int64]chan models.WebHookParsed), tp: make(map[int64]map[string]models.WebHookParsed)}, nil
}

func (b *TelegramBotApi) Start(wg *sync.WaitGroup, ctx context.Context) {
	wg.Add(1)

	go func() {
		defer wg.Done()

		b.b.Start(ctx)
	}()
}

func (b *TelegramBotApi) AskToTrackTime(wg *sync.WaitGroup, ctx context.Context) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-ctx.Done():
			fmt.Println("some")
		case val := <-b.wh:
			// fmt.Printf("Received webhook data: %+v\n", val)
			chatID := b.h.r.GetChatIDByTodoist(ctx, val.UserID)
			ch, ok := b.tq[chatID]
			if !ok {
				b.tq[chatID] = make(chan models.WebHookParsed)
				ch = b.tq[chatID]
			}
			if val.AskTime {
				ch <- val
				// add queue
				go func() {
					// TODO :: compare and swap
					for {
						if b.h.storage.GetStatus(chatID) == noActionState {

							b.h.storage.SetStatus(chatID, waitingForTimeToTrackState)
							break
						}
					}
				}()
				if b.h.storage.GetStatus(chatID) == noActionState {
					// send now
				}
				b.h.storage.SetStatus(chatID, waitingForTimeToTrackState)
			} else {
				b.b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   fmt.Sprintf("Successfully tracked: %v", val.TimeSpent),
				})
			}
		}
	}()
}
