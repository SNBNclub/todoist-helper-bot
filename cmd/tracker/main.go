package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

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
	logger.Log.Info("Starting application")

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Log.Fatal("Failed to load configuration", zap.Error(err))
	}

	logger.Log.Info("Configuration loaded successfully")

	logger.Log.Info("Process started", zap.Int("PID", os.Getpid()))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dbConfig := repository.DBConfig{
		Host:     cfg.DB_HOST,
		Port:     cfg.DB_PORT,
		DBName:   cfg.DB_NAME,
		User:     cfg.DB_USER,
		Password: cfg.DB_PASSWORD,
		MaxConns: 10,
		MaxIdle:  5,
		MaxLife:  5 * time.Minute,
	}

	dao, err := repository.New(ctx, dbConfig)
	if err != nil {
		logger.Log.Fatal("Failed to initialize database connection", zap.Error(err))
	}
	defer func() {
		if err := dao.Close(); err != nil {
			logger.Log.Error("Error closing database connection", zap.Error(err))
		}
	}()

	storage := repository.NewLocalStorage()
	defer storage.Stop()

	webhookChan := make(chan models.WebHookParsed, 10)
	authNotificationsChan := make(chan models.AuthNotification, 5)

	authHandler := handler.NewAuthHandler(cfg.APP_CLIENT_ID, cfg.APP_CLIENT_SECRET, authNotificationsChan, dao, storage)
	webhookHandler := handler.NewWebHookHandler(webhookChan)
	service := handler.NewService(authHandler, webhookHandler)

	tgBotHandlers := tgbot.NewTgHandlers(dao, storage)
	bot, err := tgbot.New(cfg.TELEGRAM_APITOKEN, dbh, tgBotHandlers, authNotificationsChan, webhookChan)
	if err != nil {
		logger.Log.Fatal("Failed to initialize Telegram bot", zap.Error(err))
	}

	wg := &sync.WaitGroup{}
	service.Start(wg, ctx)
	bot.Start(wg, ctx)

	wg.Wait()
	logger.Log.Info("Application shutdown complete")
}
