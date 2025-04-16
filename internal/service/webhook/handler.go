package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	// _ "example.com/bot/pkg/dao"

	l "example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"go.uber.org/zap"
)

func handleHTTP(w http.ResponseWriter, r *http.Request) {
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

	fmt.Printf("USER_ID: %s\n", req.UserID)

	log.Debug("Webhook request",
		zap.Any("request", req),
	)
}

func handleOAuth(w http.ResponseWriter, r *http.Request) {
	l.Log.Debug("get auth request")
	w.Write([]byte("auth page"))
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	l.Log.Debug("get main page request")
	w.Write([]byte("main page!!!"))
}

func init() {
	http.HandleFunc("/webhook", handleHTTP)
	http.HandleFunc("/auth", handleOAuth)
	http.HandleFunc("/main", handleMain)

	mux := http.NewServeMux()
	mux.HandleFunc()

	log.Fatal(http.ListenAndServe("localhost:5050", nil))
}
