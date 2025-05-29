package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"example.com/bot/internal/models"
	"example.com/bot/internal/services"
	"go.uber.org/zap"
)

type WebHookHandler struct {
	service     *services.WebHookService
	stopService func()
	logger      *zap.Logger
}

func NewWebHookHandler(service *services.WebHookService, stopService func(), logger *zap.Logger) *WebHookHandler {
	return &WebHookHandler{
		service:     service,
		stopService: stopService,
		logger:      logger.With(zap.String("handler", "WebHookHandler")),
	}
}

func (wh *WebHookHandler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	wh.logger.Debug("get webhook request")

	req := models.WebHookRequest{}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		wh.logger.Error("error while reading request body",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	logger := wh.logger.With(
		zap.String("request/body", string(body)),
	)
	if err := json.Unmarshal(body, &req); err != nil {
		// TODO :: rewrite
		// error logging level because we know all webhook formats by API docs
		// and if we can't unmarshal it - it is an error
		logger.Error("error while unmarshaling/unknown request format",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	wh.service.ProcessWebHook(&req)
	logger.Debug("webhook request handled successfully")
	w.WriteHeader(http.StatusOK)
}

func (wh *WebHookHandler) Stop() {
	wh.stopService()
}
