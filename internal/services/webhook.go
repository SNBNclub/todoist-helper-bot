package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"example.com/bot/internal/models"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const (
	itemUpdateEvent      = "item:completed"
	regexpTimeLogPattern = `^log(?P<hours_10>\d+)(?P<hours_1>\d+)(?P<mins_10>\d+)(?P<mins_1>\d+)$`
)

type WebHookService struct {
	r           *regexp.Regexp
	subexpNames []string
	wg          *sync.WaitGroup
	kafkaWriter *kafka.Writer
	logger      *zap.Logger
}

func NewWebHookService(kafkaBrokers []string, kafkaTopic string, logger *zap.Logger) (*WebHookService, func(), error) {
	r, err := regexp.Compile(regexpTimeLogPattern)
	if err != nil {
		return nil, nil, err
	}
	ws := &WebHookService{
		r:           r,
		subexpNames: r.SubexpNames(),
		wg:          &sync.WaitGroup{},
		kafkaWriter: &kafka.Writer{
			Addr:  kafka.TCP(kafkaBrokers...),
			Topic: kafkaTopic,
		},
		logger: logger.With(zap.String("service", "WebhookService")),
	}
	stop := ws.stop
	return ws, stop, nil
}

func (s *WebHookService) ProcessWebHook(wr *models.WebHookRequest) {
	s.wg.Add(1)
	go s.processWebHook(wr)
}

// TODO :: refactor to many functions
func (s *WebHookService) processWebHook(wr *models.WebHookRequest) {
	defer s.wg.Done()

	logger := s.logger.With(
		zap.String("function", "processWebHook"),
		zap.Any("event/body", wr),
	)

	logger.Debug("start webhook processing")

	if wr.EventName != itemUpdateEvent {
		logger.Debug("get not upfate event",
			zap.String("event", wr.EventName),
		)
		return
	}

	wp := models.WebHookParsed{
		UserID:    wr.UserID,
		TimeSpent: 0,
		AskTime:   false,
	}

	task := &models.Task{}
	err := json.Unmarshal(wr.EventData, &task)
	if err != nil {
		logger.Error("could not unmarshal taks entity",
			zap.Error(err),
		)
		return
	}

	wp.Task = task.Content
	if task.Duration != nil {
		switch task.Duration.Unit {
		case "minute":
			wp.TimeSpent += uint32(task.Duration.Amount)
		case "day":
			wp.TimeSpent += uint32(task.Duration.Amount) * 24 * 60
		}
		err = s.createAndSendKafkaMessage(&wp)
		if err != nil {
			logger.Error("could not send message to kafka",
				zap.Error(err),
			)
			return
		}
		logger.Debug("task with duration entity processed successfully")
		return
	}

	if len(task.Labels) == 0 {
		logger.Debug("no labels to the task/nothing to process")
		return
	}

	var matches []string
	for _, label := range task.Labels {
		if label == "track" {
			wp.AskTime = true
			err = s.createAndSendKafkaMessage(&wp)
			if err != nil {
				logger.Error("could not send message to kafka",
					zap.Error(err),
				)
				return
			}
			logger.Debug("task with label track processed successfully")
			return
		}
		matchesTmp := s.r.FindStringSubmatch(label)
		if matchesTmp != nil {
			matches = matchesTmp
			break
		}
	}

	if matches == nil {
		logger.Debug("no proper labels found for this task")
		return
	}

	result := make(map[string]uint32)
	for i, name := range s.subexpNames {
		if i > 0 && name != "" && i < len(matches) {
			dig, err := strconv.Atoi(matches[i])
			if err != nil {
				logger.Error("error while parsing finding matches",
					zap.Error(err),
				)
				return
			}
			result[name] = uint32(dig)
		}
	}

	for key, val := range result {
		switch key {
		case "hours_10":
			wp.TimeSpent += val * 600
		case "hours_1":
			wp.TimeSpent += val * 60
		case "mins_10":
			wp.TimeSpent += val * 10
		case "mins_1":
			wp.TimeSpent += val
		}
	}

	err = s.createAndSendKafkaMessage(&wp)
	if err != nil {
		logger.Error("could not send message to kafka",
			zap.Error(err),
		)
		return
	}
	logger.Debug("successfully finish webhook processing")
}

func (s *WebHookService) createAndSendKafkaMessage(wp *models.WebHookParsed) error {
	msg, err := json.Marshal(wp)
	if err != nil {
		return fmt.Errorf("could not marshal kafka message: %w", err)
	}
	return s.kafkaWriter.WriteMessages(context.Background(), kafka.Message{
		Value: msg,
	})
}

func (s *WebHookService) stop() {
	s.wg.Wait()
}
