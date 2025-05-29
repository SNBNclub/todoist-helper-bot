package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"example.com/bot/internal/models"
	"example.com/bot/internal/repository"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const (
	baseAuthURL     = "https://todoist.com/oauth/authorize"
	baseTokenGetURL = "https://todoist.com/oauth/access_token"
	SyncURL         = "https://api.todoist.com/api/v1/sync?sync_token=*&resource_types=[\"user\"]"
)

type AuthHandler struct {
	queryParams url.Values

	kakfaWriter *kafka.Writer
	r           *repository.Dao
	storage     *repository.LocalStorage
	logger      *zap.Logger
}

func NewAuthHandler(clientID, clientSecret string, kafkaBrokers []string, kafkaTopic string, r *repository.Dao, storage *repository.LocalStorage, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		queryParams: url.Values{
			"client_id":     {clientID},
			"client_secret": {clientSecret},
		},
		kakfaWriter: &kafka.Writer{
			Addr:  kafka.TCP(kafkaBrokers...),
			Topic: kafkaTopic,
		},
		r:       r,
		storage: storage,
	}
}

func (ah *AuthHandler) handleOAuth(w http.ResponseWriter, r *http.Request) {
	logger := ah.logger.With(
		zap.String("funciton", "handleOAuthRequest"),
	)
	logger.Debug("get oauth request")
	chatID, err := strconv.Atoi(r.URL.Query().Get("chat_id"))
	if err != nil {
		logger.Warn("get request without chat_id query parametr",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	logger = logger.With(
		zap.Int("chat_id", chatID),
	)
	state, err := genRandomState()
	if err != nil {
		logger.Error("unable to generate randomState",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO :: try to set cookie httponly, secure
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  state,
		Path:   "/",
		MaxAge: 300,
		// HttpOnly: true,
		// Secure: true,
	})

	ah.storage.StoreChatID(state, chatID)

	// authLink := baseAuthURL + "?" + ah.queryParams.Encode() + "&scope=data:read_write,data:delete" + "&state=" + state

	queryParams := ah.queryParams
	queryParams.Add("scope", "data:read_write,data:delete")
	queryParams.Add("state", state)

	authLink := baseAuthURL + "?" + queryParams.Encode()

	logger.Debug("redirecting to authLink",
		zap.String("authLink", authLink),
	)

	http.Redirect(w, r, authLink, http.StatusSeeOther)
}

func (ah *AuthHandler) handleCode(w http.ResponseWriter, r *http.Request) {
	logger := ah.logger.With(
		zap.String("funciton", "handleCallback"),
	)
	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		logger.Debug("can't find state cookie for request",
			zap.Error(err),
		)
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" || state != cookie.Value {
		logger.Warn("state mismatch")
		http.Error(w, "State mismatch, possible CSRF attack", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")

	queryParams := ah.queryParams
	queryParams.Add("code", code)

	url := baseTokenGetURL + "?" + queryParams.Encode()

	resp, err := http.Post(url, "", nil)
	if err != nil {
		logger.Error("could not get authorization token",
			zap.Error(err),
		)
	}
	req := models.Token{}
	defer resp.Body.Close()
	reqBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("unable to read request body",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		// TODO :: write same as webhook handler
		logger.Warn("error while unmarshaling/unknown request format",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	id, name, err := ah.getUserID(req.AccessToken)
	if err != nil {
		logger.Error("unable to get userID",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	chatID, err := ah.storage.GetChatID(state)
	if err != nil {
		logger.Error("could not get stored chatID",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = ah.r.AddTodoistUser(context.Background(), id, name)
	if err != nil {
		logger.Error("error during adding todoist user",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = ah.r.AddUserId(context.Background(), int64(chatID), id)
	if err != nil {
		logger.Error("error during adding user id",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	notification := models.AuthNotification{
		ChatID:     int64(chatID),
		Successful: true,
	}

	msgBytes, err := json.Marshal(notification)
	if err != nil {
		logger.Error("unable to marshal auth notification",
			zap.Error(err),
		)
		return
	}
	err = ah.kakfaWriter.WriteMessages(context.Background(), kafka.Message{
		Value: msgBytes,
	})
	if err != nil {
		logger.Error("unable to write kafka message",
			zap.Error(err),
		)
		return
	}
	http.Redirect(w, r, "/auth/auth_finish", http.StatusSeeOther)
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("main page!!!"))
}

func (ah *AuthHandler) getUserID(token string) (string, string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", SyncURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("uable to create http client: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer:%s", token))
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("service responded with not OK status code: %d", resp.StatusCode)
	}
	if err != nil {
		return "", "", fmt.Errorf("request done with error: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("could not read response body: %w", err)
	}
	initReq := models.InitSyncReq{}
	if err := json.Unmarshal(respBytes, &initReq); err != nil {
		return "", "", fmt.Errorf("error during unmarshaling: %w", err)
	}
	return initReq.User.ID, initReq.User.FullName, nil
}

func genRandomState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

type Service struct {
	srv *http.Server

	h *AuthHandler
	w *WebHookHandler
}

func NewService(authHandler *AuthHandler, webhookHandler *WebHookHandler) *Service {
	return &Service{
		srv: &http.Server{
			Addr: ":8080",
		},
		h: authHandler,
		w: webhookHandler,
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

	http.HandleFunc("/auth", s.h.handleOAuth)
	http.HandleFunc("/auth/callback", s.h.handleCode)
	http.HandleFunc("/webhook", s.w.handleHTTP)
	http.HandleFunc("/main", handleMain)
	http.HandleFunc("/auth/auth_finish", handleAuthFinish)

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
			// log error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()

		// TODO :: stop services

		if err := s.srv.Shutdown(ctx); err != nil {
			panic(err)
		}
	}()
}
