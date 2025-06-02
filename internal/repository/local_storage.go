package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"example.com/bot/internal/models"
	"github.com/go-redis/redis/v8"
)

type LocalStorage struct {
	client *redis.Client
}

func NewLocalStorage(ctx context.Context, addr, password string, db int) (*LocalStorage, error) {
	cl := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := cl.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("could not estabilish connection with redis db: %w", err)
	}
	return &LocalStorage{
		client: cl,
	}, nil
}

func (l *LocalStorage) StoreToken(userID, token string) error {
	return l.client.Set(context.Background(), userID, token, 0).Err()
}

func (l *LocalStorage) GetToken(userID string) (string, error) {
	return l.client.Get(context.Background(), fmt.Sprintf("token:%s", userID)).Result()
}

func (l *LocalStorage) StoreChatID(state string, chatID int) error {
	return l.client.Set(context.Background(), fmt.Sprintf("state:%s", state), chatID, 24*time.Hour).Err()
}

func (l *LocalStorage) GetChatID(state string) (int, error) {
	ctx := context.Background()
	val, err := l.client.Get(ctx, fmt.Sprintf("state:%s", state)).Int()
	// TODO ::
	// if err == redis.Nil {
	// 	return -1, nil
	// }
	if err != nil {
		return -1, err
	}
	l.client.Del(ctx, fmt.Sprintf("state:%s", state))
	return val, nil
}

func (l *LocalStorage) SetStatus(chatID int64, status int) error {
	return l.client.Set(context.Background(), fmt.Sprintf("status:%d", chatID), status, 0).Err()
}

// TODO :: redis nil
func (l *LocalStorage) GetStatus(chatID int64) (int, error) {
	val, err := l.client.Get(context.Background(), fmt.Sprintf("status:%d", chatID)).Int()
	if err == redis.Nil {
		return -1, nil
	}
	return val, err
}

func (l *LocalStorage) StoreMessageToReply(messageID int, wh models.WebHookParsed) error {
	b, err := json.Marshal(wh)
	if err != nil {
		return fmt.Errorf("error during marshaling: %w", err)
	}
	err = l.client.Set(context.Background(), fmt.Sprintf("message:%d", messageID), b, 0).Err()
	if err != nil {
		return fmt.Errorf("unable to store to redis: %w", err)
	}
	return nil
}

func (l *LocalStorage) GetMessageToReplyByID(messageID int) (*models.WebHookParsed, error) {
	key := fmt.Sprintf("message:%d", messageID)
	b, err := l.client.Get(context.Background(), key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis returned error: %w", err)
	}
	wp := models.WebHookParsed{}
	if err := json.Unmarshal([]byte(b), &wp); err != nil {
		return nil, fmt.Errorf("error during unmarshaling: %w", err)
	}
	_, err = l.client.Del(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}
	return &wp, nil
}

func (l *LocalStorage) Close() error {
	return l.client.Close()
}
