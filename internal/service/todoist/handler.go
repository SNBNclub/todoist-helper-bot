package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	tgbot "example.com/bot/internal/bot"
	"example.com/bot/internal/models"
	"example.com/bot/internal/repository"
)

const (
	baseAuthURL     = "https://todoist.com/oauth/authorize"
	baseTokenGetURL = "https://todoist.com/oauth/access_token"
	SyncURL         = "https://api.todoist.com/api/v1/sync?sync_token=*&resource_types=[\"user\"]"
)

type AuthHandler struct {
	queryParams url.Values

	r       *repository.Dao
	storage *repository.LocalStorage
}

func NewAuthHandler(clientID, clientSecret string, storage *repository.LocalStorage) *AuthHandler {
	return &AuthHandler{
		queryParams: url.Values{
			"client_id":     {clientID},
			"client_secret": {clientSecret},
		},
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

// TODO :: add chat id as request parametr to map
// use state to authentificate
// TODO :: rename a
func (a *AuthHandler) handleOAuth(w http.ResponseWriter, r *http.Request) {
	log.Println("get auth request")

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

	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  state,
		Path:   "/",
		MaxAge: 300,
	})

	a.storage.StoreState(state, chatID)

	queryParams := a.queryParams
	queryParams.Add("state", state)

	authLink := baseAuthURL + "?" + queryParams.Encode()

	http.Redirect(w, r, authLink, http.StatusSeeOther)
}

func (a *AuthHandler) handleCode(w http.ResponseWriter, r *http.Request) {
	log.Println("handle code")

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

	queryParams := a.queryParams
	queryParams.Add("code", code)

	url := baseTokenGetURL + "?" + queryParams.Encode()

	log.Println(url)

	resp, err := http.Post(url, "", nil)
	if err != nil {
		panic(err)
	}
	req := models.Token{}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&req); err != nil {
		rBytes, _ := io.ReadAll(r.Body)
		log.Fatalf("unexpected req body: %s\n", string(rBytes))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := getUserID(req.AccessToken)
	if err != nil {
		panic(err)
	}

	chatID := a.storage.GetChatID(state)
	a.r.AddUserId(context.Background(), int64(chatID), id)

	// TODO :: deeplink to tgbot redirect
	// http.Redirect(w, r, "t.me/evdocim_test_bot?regfinish=XXXX", http.StatusSeeOther)
	// TODO :: awfull, fix
	a.storage.SetStatus(int64(chatID), tgbot.TodoistRegfinishState)
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	log.Println("get main page request")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("main page!!!"))
}

func getUserID(token string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", SyncURL, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	log.Println("performing request for getting id")
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK {
		log.Println(resp.StatusCode)
		return "", nil
	}
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log.Printf("resp body: %s\n", string(res))
	user := models.SyncUser{}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		log.Println(err.Error())
	}

	return user.ID, nil
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

func (s *Service) Start(wg *sync.WaitGroup, ctx context.Context) {

	http.HandleFunc("/auth", s.h.handleOAuth)
	http.HandleFunc("/auth/callback", s.h.handleCode)
	http.HandleFunc("/webhook", s.w.handleHTTP)
	http.HandleFunc("/main", handleMain)

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
			// log error
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

}

// TODO :: or pass ctx to start
func (s *Service) Shutdown() error {
	return s.srv.Shutdown(context.Background())
}
