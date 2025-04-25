package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"example.com/bot/internal/repository"
	"go.uber.org/zap"
)

const (
	baseAuthURL     = "https://todoist.com/oauth/authorize"
	baseTokenGetURL = "https://todoist.com/oauth/access_token"
	SyncURL         = "https://api.todoist.com/api/v1/sync?sync_token=*&resource_types=[\"user\"]"
)

type AuthHandler struct {
	queryParams url.Values

	botNotifier chan<- models.AuthNotification
	r           *repository.Dao
	storage     *repository.LocalStorage
}

func NewAuthHandler(clientID, clientSecret string, botNotificatioinsChan chan<- models.AuthNotification, r *repository.Dao, storage *repository.LocalStorage) *AuthHandler {
	return &AuthHandler{
		queryParams: url.Values{
			"client_id":     {clientID},
			"client_secret": {clientSecret},
		},
		botNotifier: botNotificatioinsChan,
		r:           r,
		storage:     storage,
	}
}

func genRandomState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (ah *AuthHandler) handleOAuth(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.Atoi(r.URL.Query().Get("chat_id"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	state, err := genRandomState()
	// TODO :: status
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger.Log.Debug("state & chat_id",
		zap.String("state", state),
		zap.Int("chatID", chatID),
	)

	// TODO :: try to set cookie httponly, secure
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  state,
		Path:   "/",
		MaxAge: 300,
		// HttpOnly: true,
		// Secure: true,
	})
	logger.Log.Debug("set cookie")

	if ah.storage == nil {
		panic("storage is nill")
	}

	ah.storage.StoreState(state, chatID)

	logger.Log.Debug("state stored")

	authLink := baseAuthURL + "?" + ah.queryParams.Encode() + "&scope=data:read_write,data:delete" + "&state=" + state

	queryParams := ah.queryParams
	queryParams.Add("scope", "scope=data:read_write,data:delete")
	queryParams.Add("state", state)

	// authLink := baseAuthURL + "?" + queryParams.Encode()

	logger.Log.Debug("Auth url",
		zap.String("URL", authLink),
	)

	http.Redirect(w, r, authLink, http.StatusSeeOther)
}

func (ah *AuthHandler) handleCode(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" || state != cookie.Value {
		http.Error(w, "State mismatch, possible CSRF attack", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")

	queryParams := ah.queryParams
	queryParams.Add("code", code)

	url := baseTokenGetURL + "?" + queryParams.Encode()

	resp, err := http.Post(url, "", nil)
	if err != nil {
		panic(err)
	}
	req := models.Token{}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&req); err != nil {
		// rBytes, _ := io.ReadAll(r.Body)
		// log.Fatalf("unexpected req body: %s\n", string(rBytes))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, name, err := getUserID(req.AccessToken)
	logger.Log.Debug("data",
		zap.String("todoist_id", id),
		zap.String("todoist_name", name),
	)
	if err != nil {
		panic(err)
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
	client := &http.Client{}
	req, err := http.NewRequest("POST", SyncURL, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	// log.Println("performing request for getting id")
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK {
		// log.Println(resp.StatusCode)
		return "", "", nil
	}
	if err != nil {
		panic(err)
	}

	// res, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	panic(err)
	// }
	// log.Printf("resp body: %s\n", string(res))
	initReq := models.InitSyncReq{}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&initReq); err != nil {
		// log.Println(err.Error())
	}
	// log.Println(initReq)
	return initReq.User.ID, initReq.User.FullName, nil
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
			// log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()
		if err := s.srv.Shutdown(ctx); err != nil {
			panic(err)
		}
	}()
}
