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
	return &LocalStorage{
		tokens:        sync.Map{},
		states:        sync.Map{},
		botUserStates: sync.Map{},
	}
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
	// l.states.
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
	query, err := tools.LoadQuery("add_chat.sql")
	if err != nil {
		logger.Log.Error("Error loading SQL query",
			zap.Error(err),
		)
		return false, err
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

func (d *Dao) AddTodoistUser(ctx context.Context, todoistID string, userName string) error {
	query, err := tools.LoadQuery("add_todoist_user.sql")
	if err != nil {
		logger.Log.Error("Error loading SQL query",
			zap.Error(err),
		)
		return err
	}
	res, err := d.db.ExecContext(ctx, query, todoistID, userName)
	if err != nil {
		logger.Log.Error("error in adding todoist user",
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

func (d *Dao) AddUserId(ctx context.Context, chat_id int64, todoist_id string) error {
	query, err := tools.LoadQuery("add_chat_todoist_mapping.sql")
	if err != nil {
		logger.Log.Error("Error loading SQL query",
			zap.Error(err),
		)
		return err
	}
	res, err := d.db.ExecContext(ctx, query, chat_id, todoist_id)
	if err != nil {
		logger.Log.Error("Error in mapping todoist and tg user",
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

func (d *Dao) GetChatIDByTodoist(ctx context.Context, todoistUserID string) int64 {
	query, err := tools.LoadQuery("get_chat_id_by_todoist_id.sql")
	if err != nil {
		panic(err)
	}
	row := d.db.QueryRowContext(ctx, query, todoistUserID)
	if row.Err() != nil {
		panic(row.Err())
	}
	var chatID int64
	err = row.Scan(&chatID)
	if err != nil {
		panic(err)
	}
	return chatID
}

func (d *Dao) StoreTaskTracked(ctx context.Context, chatID int64, task models.WebHookParsed) {
	query, err := tools.LoadQuery("store_task_recording.sql")
	if err != nil {
		panic(err)
	}
	_, err = d.db.ExecContext(ctx, query, chatID, task.Task, task.TimeSpent)
	if err != nil {
		panic(err)
	}
}

func (d *Dao) GetUserStats(ctx context.Context, chatID int64) (int64, []models.TaskShow) {
	query, err := tools.LoadQuery("get_stats.sql")
	if err != nil {
		panic(err)
	}
	rows, err := d.db.Query(query, chatID)
	if err != nil {
		panic(err)
	}
	var timeSpent int64
	tasks := make([]models.TaskShow, 0, 100)
	for rows.Next() {
		tt := models.TaskShow{}
		err = rows.Scan(&timeSpent, &tt.Task, &tt.TimeSpent)
		if err != nil {
			panic(err)
		}
		tasks = append(tasks, tt)
	}
	logger.Log.Debug("get stats",
		zap.Int64("timeSpentSUm", timeSpent),
		zap.Any("task list", tasks),
	)
	return timeSpent, tasks
}

// TODO:: add error processing
func (d *Dao) Close() {
	d.db.Close()
}
