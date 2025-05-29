package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	tgbot "example.com/bot/internal/bot"
	"example.com/bot/internal/config"
	"example.com/bot/internal/handlers"
	"example.com/bot/internal/logger"
	"example.com/bot/internal/migrations"
	"example.com/bot/internal/repository"
	"example.com/bot/internal/services"
	"go.uber.org/zap"
)

func dbh(f string, args ...any) {
	log.Printf(f, args...)
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Errorf("could not load config: %w", err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logger, err := logger.NewLogger(cfg.GetLoggerConfig())
	if err != nil {
		panic(fmt.Errorf("could not create logger: %w", err))
	}

	err = migrations.RunMigrations(ctx, cfg.GetDBConnString(), cfg.DB_NAME, "file://storage/migrations")
	if err != nil {
		logger.Panic("unable to complete migrations",
			zap.Error(err),
		)
	}

	dao, err := repository.New(ctx, cfg.GetDBConnString())
	if err != nil {
		logger.Panic("unable to create dao object",
			zap.Error(err),
		)
	}

	storage, err := repository.NewLocalStorage(ctx, cfg.REDIS_HOST, cfg.REDIS_PASSWORD, cfg.REDIS_DB)
	if err != nil {
		logger.Panic("unable to create local storage",
			zap.Error(err),
		)
	}

	webHookService, stop, err := services.NewWebHookService([]string{"localhost:9092", "localhost:9093", "localhost:9094"}, "webhook", logger)
	if err != nil {
		logger.Panic("unable to create webhook service",
			zap.Error(err),
		)
	}
	authHandler := handlers.NewAuthHandler(cfg.APP_CLIENT_ID, cfg.APP_CLIENT_SECRET, []string{"localhost:9092", "localhost:9093", "localhost:9094"}, "auth", dao, storage, logger)
	webHookHandler := handlers.NewWebHookHandler(webHookService, stop, logger)

	server := handlers.NewService(authHandler, webHookHandler)

	tgBotService, err := tgbot.NewTgBotService(dao, storage, []string{"localhost:9092", "localhost:9093", "localhost:9094"}, "auth", "webhook", logger)
	if err != nil {

	}
	tgBotHandlers := tgbot.NewTgBotHandlers(tgBotService, logger)
	tgbotAPI, err := tgbot.New(cfg.TELEGRAM_APITOKEN, dbh, tgBotService, tgBotHandlers)
	if err != nil {
		logger.Panic("unable to create bot API",
			zap.Error(err),
		)
	}

	wg := sync.WaitGroup{}

	tgbotAPI.Start(&wg, ctx)
	server.Start(&wg, ctx)

	wg.Wait()
}
