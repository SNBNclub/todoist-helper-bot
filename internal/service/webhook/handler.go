package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"slices"
	"sync"

	// _ "example.com/bot/pkg/dao"

	l "example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"go.uber.org/zap"
)

const itemUpdateEvent = "item:completed"

type WebHookHandler struct {
	u chan<- models.WebHookParsed

	wg *sync.WaitGroup
}

func New(updates chan<- models.WebHookRequest) *WebHookHandler {
	return nil
}

func (wh *WebHookHandler) Start() {

}

// TODO :: add context wich listening interupts
func (wh *WebHookHandler) ShutDown() {

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
	go processWebHook(req)
	// wh.u <- req

}

func (wh *WebHookHandler) processWebHook(req *models.WebHookRequest) {
	if req.EventName != itemUpdateEvent {
		return
	}
	// TODO :: add case if task has duration
	task := req.EventData.(models.Task)

	if task.Labels == nil || len(task.Labels) == 0 {

	}

	is := slices.Contains(task.Labels, "@track")
	if is {
		// send messsage with asking how much time the task took
	}

	// TODO :: mov in init
	pattern := `^@log(?P<hours_10>\d+)(?P<hours_1>\d+):(?P<mins_10>\d+)(?P<mins_1>\d+)`
	r, err := regexp.Compile(pattern)
	if err != nil {
		panic(err)
	}

	matches := r.FindStringSubmatch(input)
	if matches == nil {
		fmt.Println("No match found")
		return
	}

	result := make(map[string]int)
	for i, name := range r.SubexpNames() {
		if i > 0 && name != "" && i < len(matches) {
			dig, err := strconv.Atoi(matches[i])
			if err != nil {
				panic(err)
			}
			result[name] = dig
		}
	}

	timeSpent := 0
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

	wp := WebHookParsed{
		UserID: req.UserID,
		TimeSpent: timeSpent,
	}

	wh.u <- wp
}

func init() {
	u := New(nil)

	http.HandleFunc("/webhook", u.handleHTTP)

	log.Fatal(http.ListenAndServe("localhost:5050", nil))
}
