package repository

import (
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/boskuv/goreminder/internal/models"
)

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
func (r *TaskRepository) CreateTask(task *models.Task) (int64, error) {
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
func (r *TaskRepository) GetTaskByID(id int64) (*models.Task, error) {
	query, args, err := r.sb.Select("id", "title", "description", "user_id", "due_date", "status", "created_at").
		From("tasks").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var task models.Task
	err = r.db.Get(&task, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task: %w", err)
	}

	return &task, nil
}
