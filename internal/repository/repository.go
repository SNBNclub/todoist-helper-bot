package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"example.com/bot/internal/logger"
	"example.com/bot/internal/models"
	"example.com/bot/pkg/tools"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Dao struct {
	db *sql.DB
}

type DBConfig struct {
	Host     string
	Port     string
	DBName   string
	User     string
	Password string
	MaxConns int
	MaxIdle  int
	MaxLife  time.Duration
}

func New(ctx context.Context, config DBConfig) (*Dao, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		config.Host, config.Port, config.DBName, config.User, config.Password,
	)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		logger.Log.Error("Failed to create database connection",
			zap.String("host", config.Host),
			zap.String("dbname", config.DBName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	if config.MaxConns > 0 {
		db.SetMaxOpenConns(config.MaxConns)
	} else {
		db.SetMaxOpenConns(10)
	}

	if config.MaxIdle > 0 {
		db.SetMaxIdleConns(config.MaxIdle)
	} else {
		db.SetMaxIdleConns(5)
	}

	if config.MaxLife > 0 {
		db.SetConnMaxLifetime(config.MaxLife)
	} else {
		db.SetConnMaxLifetime(5 * time.Minute)
	}

	if err = db.PingContext(ctx); err != nil {
		logger.Log.Error("Failed to ping database",
			zap.String("host", config.Host),
			zap.String("dbname", config.DBName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Log.Info("Database connection established successfully",
		zap.String("host", config.Host),
		zap.String("dbname", config.DBName),
	)

	return &Dao{db: db}, nil
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

func (d *Dao) GetChatIDByTodoist(ctx context.Context, todoistUserID string) (int64, error) {
	query, err := tools.LoadQuery("get_chat_id_by_todoist_id.sql")
	if err != nil {
		logger.Log.Error("Failed to load query",
			zap.String("query", "get_chat_id_by_todoist_id.sql"),
			zap.Error(err))
		return 0, fmt.Errorf("failed to load query: %w", err)
	}

	row := d.db.QueryRowContext(ctx, query, todoistUserID)
	if row.Err() != nil {
		logger.Log.Error("Database error while getting chat ID",
			zap.String("todoistUserID", todoistUserID),
			zap.Error(row.Err()))
		return 0, fmt.Errorf("database error while getting chat ID: %w", row.Err())
	}

	var chatID int64
	err = row.Scan(&chatID)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Log.Warn("No chat ID found for Todoist user",
				zap.String("todoistUserID", todoistUserID))
			return 0, nil
		}
		logger.Log.Error("Error scanning chat ID",
			zap.String("todoistUserID", todoistUserID),
			zap.Error(err))
		return 0, fmt.Errorf("error scanning chat ID: %w", err)
	}

	return chatID, nil
}

func (d *Dao) StoreTaskTracked(ctx context.Context, chatID int64, task models.WebHookParsed) error {
	query, err := tools.LoadQuery("store_task_recording.sql")
	if err != nil {
		logger.Log.Error("Failed to load SQL query",
			zap.String("query", "store_task_recording.sql"),
			zap.Error(err))
		return fmt.Errorf("failed to load query: %w", err)
	}

	_, err = d.db.ExecContext(ctx, query, chatID, task.Task, task.TimeSpent)
	if err != nil {
		logger.Log.Error("Failed to store task recording",
			zap.Int64("chatID", chatID),
			zap.String("task", task.Task),
			zap.Uint32("timeSpent", task.TimeSpent),
			zap.Error(err))
		return fmt.Errorf("failed to store task recording: %w", err)
	}

	return nil
}

func (d *Dao) GetUserStats(ctx context.Context, chatID int64) (int64, []models.TaskShow, error) {
	query, err := tools.LoadQuery("get_stats.sql")
	if err != nil {
		logger.Log.Error("Failed to load SQL query",
			zap.String("query", "get_stats.sql"),
			zap.Error(err))
		return 0, nil, fmt.Errorf("failed to load query: %w", err)
	}

	rows, err := d.db.QueryContext(ctx, query, chatID)
	if err != nil {
		logger.Log.Error("Failed to query user stats",
			zap.Int64("chatID", chatID),
			zap.Error(err))
		return 0, nil, fmt.Errorf("failed to query user stats: %w", err)
	}
	defer rows.Close()

	var timeSpent int64
	tasks := make([]models.TaskShow, 0, 100)

	for rows.Next() {
		tt := models.TaskShow{}
		err = rows.Scan(&timeSpent, &tt.Task, &tt.TimeSpent)
		if err != nil {
			logger.Log.Error("Failed to scan row data",
				zap.Int64("chatID", chatID),
				zap.Error(err))
			return 0, nil, fmt.Errorf("failed to scan row data: %w", err)
		}
		tasks = append(tasks, tt)
	}

	if err = rows.Err(); err != nil {
		logger.Log.Error("Row iteration error",
			zap.Int64("chatID", chatID),
			zap.Error(err))
		return 0, nil, fmt.Errorf("row iteration error: %w", err)
	}

	logger.Log.Debug("Retrieved user stats",
		zap.Int64("chatID", chatID),
		zap.Int64("totalTimeSpent", timeSpent),
		zap.Int("taskCount", len(tasks)),
	)

	return timeSpent, tasks, nil
}

func (d *Dao) Close() error {
	if d.db != nil {
		err := d.db.Close()
		if err != nil {
			logger.Log.Error("Error closing database connection", zap.Error(err))
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}
	return nil
}

const (
	NoActionState = iota
	TodoistRegisteringState
	WaitingForTimeToTrackState
)
