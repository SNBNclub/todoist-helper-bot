package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"example.com/bot/pkg/tools"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type LocalStorage struct {
	tokens        sync.Map
	states        sync.Map
	botUserStates sync.Map
}

func NewLocalStorage() *LocalStorage {
	return &LocalStorage{}
}

func (l *LocalStorage) StoreToken(userID, token string) {
	l.tokens.Store(userID, token)
}

func (l *LocalStorage) GetToken(userID string) string {
	val, ok := l.tokens.Load(userID)
	if !ok {
		return ""
	}
	return val.(string)
}

func (l *LocalStorage) StoreState(state string, chatID int) {
	l.states.Store(state, chatID)
}

func (l *LocalStorage) GetChatID(state string) int {
	val, loaded := l.states.LoadAndDelete(state)
	if !loaded {
		return -1
	}
	return val.(int)
}

func (l *LocalStorage) SetStatus(chatID int64, status int) {
	l.botUserStates.Store(chatID, status)
}

func (l *LocalStorage) GetStatus(chatID int64) int {
	val, ok := l.botUserStates.Load(chatID)
	if !ok {
		return -1
	}
	return val.(int)
}

type Dao struct {
	db *sql.DB
}

func New(ctx context.Context, host, port, dbName, user, password string) *Dao {
	constring := fmt.Sprintf("host=%s port=%s database=%s user=%s password=%s", host, port, dbName, user, password)
	db, err := sql.Open("pgx", constring)

	if err != nil {
		logger.Log.Error("Error in creating database connection",
			zap.String("dsn", constring),
			zap.Error(err),
		)
	}
	if err = db.PingContext(ctx); err != nil {
		logger.Log.Error("Error while pinging database",
			zap.Error(err),
		)
		db.Close()
		return nil
	}
	logger.Log.Debug("Creating dao object successfylly")
	return &Dao{db: db}
}

func (d *Dao) CreateUser(ctx context.Context, u *models.TgUser) (bool, error) {
	query, err := tools.LoadQuery("sql/add_chat.sql")
	if err != nil {

	}
	var isNewUser bool
	err = d.db.QueryRowContext(ctx, query, u.ChatID, u.Name).Scan(&isNewUser)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		logger.Log.Error("Error in user creating",
			zap.Error(err),
		)
		return false, err
	}
	return true, nil
}

func (d *Dao) AddUserId(ctx context.Context, chat_id int64, todoist_id string) error {
	query, err := tools.LoadQuery("add_chat_todoist_mapping.sql")
	if err != nil {

	}
	res, err := d.db.ExecContext(ctx, query, chat_id, todoist_id)
	if err != nil {
		logger.Log.Error("Error in user creating",
			zap.Error(err),
		)
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Log.Error("Error while checking affected rows",
			zap.Error(err),
		)
		return err
	}
	if n != 1 {
		logger.Log.Warn("Unexpected rows affected while user creating",
			zap.Int64("affected", n),
		)
		return err
	}
	return nil
}

// TODO:: add error processing
func (d *Dao) Close() {
	d.db.Close()
}
