package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	l "example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"go.uber.org/zap"
)

const (
	itemUpdateEvent      = "item:completed"
	regexpTimeLogPattern = `^log(?P<hours_10>\d+)(?P<hours_1>\d+)(?P<mins_10>\d+)(?P<mins_1>\d+)$`
)

type WebHookHandler struct {
	u           chan<- models.WebHookParsed
	r           *regexp.Regexp
	subexpNames []string
	wg          *sync.WaitGroup
}

func NewWebHookHandler(updates chan<- models.WebHookParsed) *WebHookHandler {
	r := regexp.MustCompile(regexpTimeLogPattern)
	return &WebHookHandler{
		u:           updates,
		r:           r,
		subexpNames: r.SubexpNames(),
		wg:          &sync.WaitGroup{},
	}
}

func (wh *WebHookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wh.handleHTTP(w, r)
}

func (wh *WebHookHandler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	log := l.Log.With(
		zap.String("host", r.Host),
		zap.String("method", r.Method),
		zap.String("url", r.URL.String()),
		zap.String("remote_addr", r.RemoteAddr),
	)
	log.Debug("Received webhook request")

	if r.Method != http.MethodPost {
		log.Warn("Invalid request method", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	defer r.Body.Close()

	if err != nil {
		log.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	req := models.WebHookRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		log.Error("Failed to unmarshal request body",
			zap.Error(err),
			zap.ByteString("body", body),
		)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.EventName == "" || req.UserID == "" {
		log.Warn("Required fields missing in webhook payload",
			zap.Any("request", req),
		)
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	wh.wg.Add(1)
	go func() {
		defer wh.wg.Done()
		wh.processWebHook(&req)
	}()

	duration := time.Since(startTime)
	log.Debug("Webhook request processed",
		zap.Duration("duration", duration),
		zap.String("event", req.EventName),
		zap.String("user", req.UserID),
	)

	w.WriteHeader(http.StatusOK)
}

func (wh *WebHookHandler) processWebHook(req *models.WebHookRequest) {
	log := l.Log.With(
		zap.String("event", req.EventName),
		zap.String("user_id", req.UserID),
	)

	if req.EventName != itemUpdateEvent {
		log.Debug("Ignoring non-completed item event")
		return
	}

	wp := models.WebHookParsed{
		UserID:    req.UserID,
		TimeSpent: 0,
		AskTime:   false,
	}

	task, err := req.GetTaskData()
	if err != nil {
		log.Error("Failed to parse task data", zap.Error(err))
		return
	}

	wp.Task = task.Content
	log.Debug("Processing completed task", zap.String("task", wp.Task))

	// TODO :: separate function to calculate with duration
	if task.Duration != nil {
		log.Debug("Task has duration specified",
			zap.Int("amount", task.Duration.Amount),
			zap.String("unit", task.Duration.Unit),
		)

		switch task.Duration.Unit {
		case "minute":
			wp.TimeSpent += uint32(task.Duration.Amount)
		case "day":
			wp.TimeSpent += uint32(task.Duration.Amount) * 24 * 60
		case "hour":
			wp.TimeSpent += uint32(task.Duration.Amount) * 60
		default:
			log.Warn("Unknown duration unit", zap.String("unit", task.Duration.Unit))
		}

		if wp.TimeSpent > 0 {
			log.Info("Sending webhook with duration",
				zap.Uint32("timeSpent", wp.TimeSpent),
				zap.String("unit", task.Duration.Unit),
				zap.String("task", wp.Task),
			)
			wh.u <- wp
			return
		}
	}

	if len(task.Labels) == 0 {
		log.Debug("Task has no labels to process")
		return
	}

	var matches []string
	for _, label := range task.Labels {
		if label == "@track" {
			log.Info("Found @track label, asking for time tracking")
			wp.AskTime = true
			wh.u <- wp
			return
		}

		matchesTmp := wh.r.FindStringSubmatch(label)
		if matchesTmp != nil {
			log.Debug("Found time log label", zap.String("label", label))
			matches = matchesTmp
			break
		}
	}

	if matches == nil {
		log.Debug("No time tracking labels found")
		return
	}

	result, err := wh.extractTimeComponents(matches)
	if err != nil {
		log.Error("Failed to extract time components", zap.Error(err))
		return
	}

	wp.TimeSpent = wh.calculateTimeSpent(result)

	log.Info("Time spent calculated from label",
		zap.Uint32("timeSpent", wp.TimeSpent),
		zap.Any("components", result),
		zap.String("task", wp.Task),
	)

	wh.u <- wp
}

func (wh *WebHookHandler) extractTimeComponents(matches []string) (map[string]uint32, error) {
	result := make(map[string]uint32)

	for i, name := range wh.subexpNames {
		if i > 0 && name != "" && i < len(matches) {
			dig, err := strconv.Atoi(matches[i])
			if err != nil {
				return nil, fmt.Errorf("failed to parse time digit '%s' for '%s': %w",
					matches[i], name, err)
			}
			result[name] = uint32(dig)
		}
	}

	return result, nil
}

func (wh *WebHookHandler) calculateTimeSpent(components map[string]uint32) uint32 {
	var timeSpent uint32 = 0

	for key, val := range components {
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

	return timeSpent
}
