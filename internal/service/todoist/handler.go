package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"example.com/bot/internal/repository"
	"go.uber.org/zap"
)

const (
	baseAuthURL     = "https://todoist.com/oauth/authorize"
	baseTokenGetURL = "https://todoist.com/oauth/access_token"
	syncURL         = "https://api.todoist.com/api/v1/sync?sync_token=*&resource_types=[\"user\"]"
	defaultScope    = "data:read_write,data:delete"
	authTimeout     = 15 * time.Minute
	cookieMaxAge    = 900
)

type AuthNotificationType int

const (
	AuthSuccess AuthNotificationType = iota
	AuthTimeout
	AuthError
)

type AuthHandler struct {
	queryParams  url.Values
	botNotifier  chan<- models.AuthNotification
	r            *repository.Dao
	storage      *repository.LocalStorage
	authTimeout  time.Duration
	secureCookie bool
}

func NewAuthHandler(clientID, clientSecret string, botNotificationsChan chan<- models.AuthNotification, r *repository.Dao, storage *repository.LocalStorage) *AuthHandler {
	return &AuthHandler{
		queryParams: url.Values{
			"client_id":     {clientID},
			"client_secret": {clientSecret},
		},
		botNotifier:  botNotificationsChan,
		r:            r,
		storage:      storage,
		authTimeout:  authTimeout,
		secureCookie: true,
	}
}

func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (ah *AuthHandler) handleOAuth(w http.ResponseWriter, r *http.Request) {
	log := logger.Log.With(
		zap.String("handler", "handleOAuth"),
		zap.String("remote_addr", r.RemoteAddr),
	)

	chatIDStr := r.URL.Query().Get("chat_id")
	if chatIDStr == "" {
		log.Warn("Missing chat_id parameter")
		http.Error(w, "Missing chat_id parameter", http.StatusBadRequest)
		return
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		log.Error("Invalid chat_id parameter",
			zap.String("chat_id", chatIDStr),
			zap.Error(err),
		)
		http.Error(w, "Invalid chat_id parameter", http.StatusBadRequest)
		return
	}

	state, err := generateRandomState()
	if err != nil {
		log.Error("Failed to generate random state", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Debug("Generated OAuth state",
		zap.String("state", state),
		zap.Int64("chatID", chatID),
	)

	ah.storage.StoreState(state, int(chatID))

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		Secure:   ah.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	queryParams := ah.queryParams
	queryParams.Add("scope", defaultScope)
	queryParams.Add("state", state)

	authURL := baseAuthURL + "?" + queryParams.Encode()

	log.Info("Redirecting to Todoist OAuth",
		zap.String("auth_url", authURL),
		zap.Int64("chat_id", chatID),
		zap.String("state", state),
	)

	go ah.setAuthTimeout(state, chatID)

	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func (ah *AuthHandler) setAuthTimeout(state string, chatID int64) {
	time.Sleep(ah.authTimeout)
	if ah.storage.IsStateValid(state, int(chatID)) {
		ah.botNotifier <- models.AuthNotification{
			ChatID:     chatID,
			Successful: false,
			Error:      errors.New("authentication timeout"),
			Type:       int(AuthTimeout),
		}
		ah.storage.InvalidateState(state, int(chatID))
	}
}

func (ah *AuthHandler) handleCode(w http.ResponseWriter, r *http.Request) {
	log := logger.Log.With(
		zap.String("handler", "handleCode"),
		zap.String("remote_addr", r.RemoteAddr),
	)

	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		log.Error("State cookie not found", zap.Error(err))
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" || state != cookie.Value {
		log.Error("State mismatch, possible CSRF attack")
		http.Error(w, "State mismatch, possible CSRF attack", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")

	queryParams := ah.queryParams
	queryParams.Add("code", code)

	url := baseTokenGetURL + "?" + queryParams.Encode()

	resp, err := http.Post(url, "", nil)
	if err != nil {
		log.Error("Failed to exchange code for token", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	req := models.Token{}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&req); err != nil {
		rBytes, _ := io.ReadAll(resp.Body)
		log.Error("Unexpected request body",
			zap.String("body", string(rBytes)),
			zap.Error(err))
		http.Error(w, "Failed to decode token response", http.StatusInternalServerError)
		return
	}

	id, name, err := getUserID(req.AccessToken)
	logger.Log.Debug("data",
		zap.String("todoist_id", id),
		zap.String("todoist_name", name),
	)
	if err != nil {
		log.Error("Failed to get user ID", zap.Error(err))
		http.Error(w, "Failed to get user ID", http.StatusInternalServerError)
		return
	}

	chatID := ah.storage.GetChatID(state)
	logger.Log.Debug("chat_ID",
		zap.Int("chatID", chatID),
	)
	ah.r.AddTodoistUser(context.Background(), id, name)
	ah.r.AddUserId(context.Background(), int64(chatID), id)

	ah.botNotifier <- models.AuthNotification{
		ChatID:     int64(chatID),
		Successful: true,
	}
	http.Redirect(w, r, "/auth/auth_finish", http.StatusSeeOther)
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("main page!!!"))
}

func getUserID(token string) (string, string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", syncURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var initReq models.InitSyncReq
	if err := json.NewDecoder(resp.Body).Decode(&initReq); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	if initReq.User.ID == "" {
		return "", "", errors.New("user ID not found in response")
	}

	return initReq.User.ID, initReq.User.FullName, nil
}

type Service struct {
	srv *http.Server

	h *AuthHandler
	w *WebHookHandler
}

func NewService(authHandler *AuthHandler, webhookHandler *WebHookHandler) *Service {
	srv := &http.Server{
		Addr: ":8080",
	}

	return &Service{
		h:   authHandler,
		w:   webhookHandler,
		srv: srv,
	}
}

// TODO :: hide auth finish page
func handleAuthFinish(w http.ResponseWriter, r *http.Request) {
	html := `
    <!DOCTYPE html>
    <html>
    <head>
        <title>Authentication Successful</title>
        <style>
            body { font-family: Arial, sans-serif; text-align: center; margin-top: 50px; }
            .success { color: #4CAF50; font-size: 24px; margin-bottom: 20px; }
            .message { font-size: 18px; margin-bottom: 30px; }
            .button { background-color: #4CAF50; color: white; padding: 10px 20px; 
                     text-decoration: none; border-radius: 5px; font-size: 16px; }
        </style>
    </head>
    <body>
        <div class="success">Authentication Successful</div>
        <div class="message">Your Todoist account has been linked successfully.</div>
        <a class="button" href="https://t.me/evdocim_test_bot">Return to Bot</a>
    </body>
    </html>
    `
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *Service) Start(wg *sync.WaitGroup, ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", s.h.handleOAuth)
	mux.HandleFunc("/auth/callback", s.h.handleCode)
	mux.HandleFunc("/webhook", s.w.handleHTTP)
	mux.HandleFunc("/main", handleMain)
	mux.HandleFunc("/auth/auth_finish", handleAuthFinish)

	s.srv.Handler = mux

	s.srv.ReadTimeout = 10 * time.Second
	s.srv.WriteTimeout = 10 * time.Second
	s.srv.IdleTimeout = 120 * time.Second

	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Log.Info("Starting HTTP server", zap.String("address", s.srv.Addr))

		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("HTTP server error", zap.Error(err))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()
		logger.Log.Info("Shutting down HTTP server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			logger.Log.Error("HTTP server shutdown error", zap.Error(err))
		}

		logger.Log.Info("HTTP server stopped")
	}()
}
