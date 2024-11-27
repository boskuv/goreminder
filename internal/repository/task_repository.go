// internal/repository/task_repository.go
package repository

import (
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type Task struct {
	ID          int64     `db:"id"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	UserID      int64     `db:"user_id"`
	DueDate     time.Time `db:"due_date"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
}

type TaskRepository struct {
	db *sqlx.DB
	sb squirrel.StatementBuilderType
}

func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{
		db: db,
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar), // Use PostgreSQL dollar placeholders
	}
}

// CreateTask inserts a new task into the database
func (r *TaskRepository) CreateTask(task *Task) (int64, error) {
	query, args, err := r.sb.Insert("tasks").
		Columns("title", "description", "user_id", "due_date", "status").
		Values(task.Title, task.Description, task.UserID, task.DueDate, task.Status).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build query: %w", err)
	}

	var id int64
	err = r.db.QueryRow(query, args...).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert task: %w", err)
	}

	return id, nil
}

// GetTaskByID retrieves a task by its ID
func (r *TaskRepository) GetTaskByID(id int64) (*Task, error) {
	query, args, err := r.sb.Select("id", "title", "description", "user_id", "due_date", "status", "created_at").
		From("tasks").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var task Task
	err = r.db.Get(&task, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task: %w", err)
	}

	return &task, nil
}
