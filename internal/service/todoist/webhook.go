package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"

	"example.com/bot/internal/logger"
	l "example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"go.uber.org/zap"
)

const (
	itemUpdateEvent      = "item:completed"
	regexpTimeLogPattern = `^log(?P<hours_10>\d+)(?P<hours_1>\d+)(?P<mins_10>\d+)(?P<mins_1>\d+)$`
)

type WebHookHandler struct {
	u chan<- models.WebHookParsed

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

func (wh *WebHookHandler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	log := l.Log.With(
		zap.String("host", r.Host),
		zap.String("method", r.Method),
		zap.String("url", r.URL.String()),
		zap.String("remote address", r.RemoteAddr),
	)
	log.Debug("Recieve webhook request")
	req := models.WebHookRequest{}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rBytes, _ := io.ReadAll(r.Body)
		log.Error("Unexpecter request body",
			zap.String("body", string(rBytes)),
		)
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}

	w.WriteHeader(http.StatusOK)

	log.Debug("Webhook request",
		zap.Any("request", req),
	)

	wh.wg.Add(1)
	go wh.processWebHook(&req)
	// wh.u <- req

}

func (wh *WebHookHandler) processWebHook(req *models.WebHookRequest) {
	defer wh.wg.Done()

	if req.EventName != itemUpdateEvent {
		logger.Log.Debug("wtf")
		return
	}

	wp := models.WebHookParsed{
		UserID:    req.UserID,
		TimeSpent: 0,
		AskTime:   false,
	}

	task := &models.Task{}
	err := json.Unmarshal(req.EventData, &task)
	if err != nil {
		logger.Log.Error("error in unmarshaling",
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
		logger.Log.Debug("wirte to chan")
		wh.u <- wp
		return
	}

	if len(task.Labels) == 0 {
		logger.Log.Debug("impossible")
		return
	}

	var matches []string
	for _, label := range task.Labels {
		if label == "track" {
			wp.AskTime = true
			logger.Log.Debug("wirte to chan")
			wh.u <- wp
			return
		}
		matchesTmp := wh.r.FindStringSubmatch(label)
		if matchesTmp != nil {
			matches = matchesTmp
			break
		}
	}

	if matches == nil {
		return
	}

	result := make(map[string]uint32)
	for i, name := range wh.subexpNames {
		if i > 0 && name != "" && i < len(matches) {
			dig, err := strconv.Atoi(matches[i])
			if err != nil {
				panic(err)
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
	logger.Log.Debug("wirte to chan")
	wh.u <- wp
}
