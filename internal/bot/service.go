package tgbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"example.com/bot/internal/models"
	"example.com/bot/internal/repository"
	"go.uber.org/zap"

	"github.com/go-redis/redis/v8"
	"github.com/go-telegram/bot"
	m "github.com/go-telegram/bot/models"
	"github.com/segmentio/kafka-go"
)

var ErrUncorrectTimeFromat = errors.New("uncorrect time foramt")

const (
	pattern       = `^(?P<hours_10>\d+)(?P<hours_1>\d+)(?P<mins_10>\d+)(?P<mins_1>\d+)`
	noActionState = iota
	todoistRegisteringState
	waitingForTimeToTrackState
)

type TgBotService struct {
	regexp                       *regexp.Regexp
	repository                   repository.Dao
	storage                      *repository.LocalStorage
	kafkaReaderAuthNotifications *kafka.Reader
	kafkaReaderWebHooks          *kafka.Reader
	logger                       *zap.Logger
}

func NewTgBotService(repository *repository.Dao, storage *repository.LocalStorage, kafkaBrokers []string, kafkaAuthTopic, kafkaWebHookTopic string, logger *zap.Logger) (*TgBotService, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &TgBotService{
		regexp:     r,
		repository: *repository,
		storage:    storage,
		kafkaReaderAuthNotifications: kafka.NewReader(kafka.ReaderConfig{
			Brokers: kafkaBrokers,
			Topic:   kafkaAuthTopic,
		}),
		kafkaReaderWebHooks: kafka.NewReader(kafka.ReaderConfig{
			Brokers: kafkaBrokers,
			Topic:   kafkaWebHookTopic,
		}),
		logger: logger,
	}, nil
}

func (s *TgBotService) listenAuthNotifications(ctx context.Context, b *bot.Bot) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := s.kafkaReaderAuthNotifications.ReadMessage(ctx)
			if err != nil {
				s.logger.Error("error reading kafka message",
					zap.Error(err),
				)
				continue
			}
			// TODO :: process error and send message
			var wp models.AuthNotification
			if err := json.Unmarshal(msg.Value, &wp); err != nil {
				s.logger.Error("error during unmarshaling kafka message",
					zap.Error(err),
				)
				continue
			}
			sendMessageParams := &bot.SendMessageParams{
				ChatID: wp.ChatID,
			}
			if wp.Successful {
				sendMessageParams.Text = "Todoist authentication completed successfully!"
			} else {
				sendMessageParams.Text = "Todoist authentication failed. Please try again."
			}
			err = s.storage.SetStatus(wp.ChatID, noActionState)
			if err != nil {
				s.logger.Error("could not set status for user",
					zap.Error(err),
				)
				continue
			}
			_, err = b.SendMessage(ctx, sendMessageParams)
			if err != nil {
				s.logger.Error("error during sending message",
					zap.Error(err),
				)
			}
		}
	}
}

func (s *TgBotService) listenWebhooks(ctx context.Context, b *bot.Bot) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := s.kafkaReaderWebHooks.ReadMessage(ctx)
			if err != nil {
				s.logger.Error("error reading kafka message",
					zap.Error(err),
				)
				continue
			}
			// TODO :: process error and send message
			var wp models.WebHookParsed
			if err := json.Unmarshal(msg.Value, &wp); err != nil {
				s.logger.Error("error during unmarshaling kafka message",
					zap.Error(err),
				)
				continue
			}
			// TODO :: process error and send message
			chatID, err := s.repository.GetChatIDByTodoist(ctx, wp.UserID)
			if err != nil {
				s.logger.Error("error trying to get chatID by todoistID",
					zap.Error(err),
				)
				continue
			}
			sendMessageParams := &bot.SendMessageParams{
				ChatID: chatID,
			}
			if !wp.AskTime {
				err = s.repository.StoreTaskTracked(ctx, chatID, wp)
				if err != nil {
					s.logger.Error("error during storing tracked taks",
						zap.Error(err),
					)
					sendMessageParams.Text = "service error try again later"
				} else {
					sendMessageParams.Text = fmt.Sprintf("Stored %d for task: %s", wp.TimeSpent, wp.Task)
				}
			} else {
				sendMessageParams.Text = fmt.Sprintf("Enter time for this task in reply message: %s", wp.Task)
			}
			msgSend, err := b.SendMessage(ctx, sendMessageParams)
			if err != nil {
				s.logger.Error("error during sending message",
					zap.Error(err),
				)
				continue
			}
			err = s.storage.StoreMessageToReply(msgSend.ID, wp)
			if err != nil {
				s.logger.Error("error during storing message to reply",
					zap.Error(err),
				)
			}
		}
	}
}

func (s *TgBotService) processStart(ctx context.Context, update *m.Update) *bot.SendMessageParams {
	logger := s.logger.With(
		zap.String("TgBotService", "processStart"),
		zap.Any("update", update),
	)
	if update.Message == nil {
		logger.Debug("no message update")
		return nil
	}
	u := &models.TgUser{
		ChatID: update.Message.Chat.ID,
		Name:   update.Message.Chat.Username,
	}
	sendMessageParams := &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
	}
	isNewUser, err := s.repository.CreateUser(ctx, u)
	if err != nil {
		logger.Error("error during user creating",
			zap.Error(err),
		)
	}
	if !isNewUser {
		// TODO ::
		sendMessageParams.Text = "Welocme back...!"
		return sendMessageParams
	}
	sendMessageParams.Text = "Welcome!"
	return sendMessageParams
}

func (s *TgBotService) processStats(ctx context.Context, update *m.Update) *bot.SendMessageParams {
	logger := s.logger.With(
		zap.String("TgBotService", "processStats"),
		zap.Any("update", update),
	)
	chatID := update.Message.Chat.ID
	sendMessageParams := &bot.SendMessageParams{
		ChatID: chatID,
	}
	timeSpent, tasks, err := s.repository.GetUserStats(ctx, chatID)
	if err != nil {
		logger.Error("error trying to get user stats",
			zap.Error(err),
		)
		sendMessageParams.Text = "Service error, try again later"
		return sendMessageParams
	}
	message := fmt.Sprintf("You spent: %d\n", timeSpent)
	for _, t := range tasks {
		message += fmt.Sprintf("Task: %s - %d\n", t.Task, t.TimeSpent)
	}
	sendMessageParams.Text = message
	return sendMessageParams
}

func (s *TgBotService) processAuth(update *m.Update) *bot.SendMessageParams {
	logger := s.logger.With(
		zap.String("TgBotService", "processAuth"),
	)
	chatID := update.Message.Chat.ID
	ch := strconv.FormatInt(chatID, 10)
	link := "https://snbn.online/auth?chat_id=" + ch
	err := s.storage.SetStatus(chatID, todoistRegisteringState)
	if err != nil {
		logger.Error("could not set status for user",
			zap.Error(err),
		)
	}
	return &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("Auth using this [link](%s)", link),
		ParseMode: m.ParseModeMarkdown,
	}
}

func (s *TgBotService) processUpdate(ctx context.Context, update *m.Update) *bot.SendMessageParams {
	logger := s.logger.With(
		zap.String("TgBotService", "processUpdate"),
		zap.Any("update", update),
	)
	logger.Debug("service received update")
	if update.Message == nil {
		logger.Debug("not message update")
		return nil
	}
	if update.Message.ReplyToMessage != nil {
		return s.processReply(ctx, update)
	}
	return s.processMessage(update)
}

func (s *TgBotService) processReply(ctx context.Context, update *m.Update) *bot.SendMessageParams {
	logger := s.logger.With(
		zap.String("TgBotService", "processReply"),
	)
	logger.Debug("processing reply")
	chatID := update.Message.Chat.ID
	sendMessageParams := &bot.SendMessageParams{
		ChatID: chatID,
	}
	if update.Message.Text == "/ignore_task" {
		sendMessageParams.Text = "OK, no tracking for this task"
		return sendMessageParams
	}
	wp, err := s.storage.GetMessageToReplyByID(update.Message.ReplyToMessage.ID)
	if err != nil && err != redis.Nil {
		logger.Error("error trying to get message data",
			zap.Error(err),
		)
		sendMessageParams.Text = "Server error, try again later"
		return sendMessageParams
	}
	if err == redis.Nil {
		logger.Debug("reply is useless")
		return nil
	}
	timeSpent, err := s.processMessageText(update.Message.Text)
	if err != nil && err != ErrUncorrectTimeFromat {
		logger.Error("error during processing message text",
			zap.Error(err),
		)
		sendMessageParams.Text = "Server error, try again later"
		return sendMessageParams
	}
	if err == ErrUncorrectTimeFromat {
		logger.Debug("get message with uncorrect time format",
			zap.String("message", update.Message.Text),
		)
		sendMessageParams.Text = "Uncorrect time format"
		return sendMessageParams
	}
	wp.TimeSpent = timeSpent
	err = s.repository.StoreTaskTracked(ctx, chatID, *wp)
	if err != nil {
		logger.Error("could not store tracked task",
			zap.Error(err),
		)
		return nil
	}
	sendMessageParams.Text = fmt.Sprintf("Task: %s succesfully tracked: %d", wp.Task, wp.TimeSpent)
	return sendMessageParams
}

func (s *TgBotService) processMessage(update *m.Update) *bot.SendMessageParams {
	logger := s.logger.With(
		zap.String("TgBotService", "processMessage"),
	)
	state, err := s.storage.GetStatus(update.Message.Chat.ID)
	if err != nil {
		logger.Error("error trying to get user status",
			zap.Error(err),
		)
		return nil
	}
	sendMessageParams := &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
	}
	switch state {
	case todoistRegisteringState:
		if update.Message.Text == "/cancel" {
			sendMessageParams.Text = "Finish reg later"
		} else {
			sendMessageParams.Text = "Use /cancel or finish the reg"
		}
	}
	return sendMessageParams
}

func (s *TgBotService) processMessageText(messageText string) (uint32, error) {
	matches := s.regexp.FindStringSubmatch(messageText)
	if matches == nil {
		return 0, ErrUncorrectTimeFromat
	}
	result := make(map[string]uint32)
	for i, name := range s.regexp.SubexpNames() {
		if i > 0 && name != "" && i < len(matches) {
			dig, err := strconv.Atoi(matches[i])
			if err != nil {
				return 0, err
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
	return timeSpent, nil
}
