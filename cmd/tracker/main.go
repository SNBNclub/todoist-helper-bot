package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"

	config "example.com/bot/configs"
	tgbot "example.com/bot/internal/bot"
	"example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"example.com/bot/internal/repository"
	handler "example.com/bot/internal/service/todoist"

	"go.uber.org/zap"
)

func dbh(f string, args ...any) {
	log.Printf(f, args...)
}

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

	r := repository.New(ctx, cfg.DB_HOST, cfg.DB_PORT, cfg.DB_NAME, cfg.DB_USER, cfg.DB_PASSWORD)
	storage := repository.NewLocalStorage()
	ch := make(chan models.WebHookParsed)

	ah := handler.NewAuthHandler(cfg.APP_CLIENT_ID, cfg.APP_CLIENT_SECRET, storage)
	wh := handler.NewWebHookHandler(ch)
	srv := handler.NewService(ah, wh)

	tgBotHandlers := tgbot.NewTgHandlers(r, storage)
	b, err := tgbot.New(cfg.TELEGRAM_APITOKEN, dbh, tgBotHandlers)
	if err != nil {
		panic(err)
	}

	wg := &sync.WaitGroup{}

	srv.Start(wg, ctx)
	b.Start(wg, ctx)

	wg.Wait()
}
