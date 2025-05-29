package repository

import (
	"context"
	"database/sql"
	"fmt"

	"example.com/bot/internal/models"
	"example.com/bot/tools"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Dao struct {
	db *sql.DB
}

func New(ctx context.Context, connString string) (*Dao, error) {
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, fmt.Errorf("could not open connection with database: %w", err)
	}
	if err = db.PingContext(ctx); err != nil {
		// TODO :: will it work
		err = fmt.Errorf("could not estabilish connection with databse: %w", err)
		err = fmt.Errorf("while closing db: %w %s", db.Close(), err.Error())
		return nil, err
	}
	return &Dao{db: db}, nil
}

func (d *Dao) CreateUser(ctx context.Context, u *models.TgUser) (bool, error) {
	query, err := tools.LoadQuery("add_chat.sql")
	if err != nil {
		return false, fmt.Errorf("could not load query: %w", err)
	}
	var isNewUser bool
	err = d.db.QueryRowContext(ctx, query, u.ChatID, u.Name).Scan(&isNewUser)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("error during executing query: %w", err)
	}
	return true, nil
}

func (d *Dao) AddTodoistUser(ctx context.Context, todoistID string, userName string) error {
	query, err := tools.LoadQuery("add_todoist_user.sql")
	if err != nil {
		return fmt.Errorf("could not load query: %w", err)
	}
	res, err := d.db.ExecContext(ctx, query, todoistID, userName)
	if err != nil {
		return fmt.Errorf("error during executing query: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected returns error: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("unexpected amount of rows affected: %w", err)
	}
	return nil
}

func (d *Dao) AddUserId(ctx context.Context, chat_id int64, todoist_id string) error {
	query, err := tools.LoadQuery("add_chat_todoist_mapping.sql")
	if err != nil {
		return fmt.Errorf("could not load query: %w", err)
	}
	res, err := d.db.ExecContext(ctx, query, chat_id, todoist_id)
	if err != nil {
		return fmt.Errorf("error during executing query: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected returns error: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("unexpected amount of rows affected: %w", err)
	}
	return nil
}

func (d *Dao) GetChatIDByTodoist(ctx context.Context, todoistUserID string) (int64, error) {
	query, err := tools.LoadQuery("get_chat_id_by_todoist_id.sql")
	if err != nil {
		return -1, fmt.Errorf("could not load query: %w", err)
	}
	row := d.db.QueryRowContext(ctx, query, todoistUserID)
	if row.Err() != nil {
		return -1, fmt.Errorf("error during executing query: %w", err)
	}
	var chatID int64
	err = row.Scan(&chatID)
	if err != nil {
		return -1, fmt.Errorf("error while scanning result: %w", err)
	}
	return chatID, nil
}

func (d *Dao) StoreTaskTracked(ctx context.Context, chatID int64, task models.WebHookParsed) error {
	query, err := tools.LoadQuery("store_task_recording.sql")
	if err != nil {
		return fmt.Errorf("could not load query: %w", err)
	}
	_, err = d.db.ExecContext(ctx, query, chatID, task.Task, task.TimeSpent)
	if err != nil {
		return fmt.Errorf("error during executing query: %w", err)
	}
	return nil
}

func (d *Dao) GetUserStats(ctx context.Context, chatID int64) (int64, []models.TaskShow, error) {
	query, err := tools.LoadQuery("get_stats.sql")
	if err != nil {
		return -1, nil, fmt.Errorf("could not load query: %w", err)
	}
	rows, err := d.db.Query(query, chatID)
	if err != nil {
		return -1, nil, fmt.Errorf("error during executing query: %w", err)
	}
	var timeSpent int64
	tasks := make([]models.TaskShow, 0, 100)
	for rows.Next() {
		tt := models.TaskShow{}
		err = rows.Scan(&timeSpent, &tt.Task, &tt.TimeSpent)
		if err != nil {
			return -1, nil, fmt.Errorf("error while scanning result: %w", err)
		}
		tasks = append(tasks, tt)
	}
	return timeSpent, tasks, nil
}

// TODO:: add error processing
func (d *Dao) Close() {
	d.db.Close()
}
