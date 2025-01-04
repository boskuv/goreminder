package repository

import (
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

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
		Columns("title", "description", "user_id", "due_date", "status", "deleted_at").
		Values(task.Title, task.Description, task.UserID, task.DueDate, task.Status, nil).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query")
	}

	var id int64
	err = r.db.QueryRow(query, args...).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert task")
	}

	return id, nil
}

// GetTaskByID retrieves a task by its ID
func (r *TaskRepository) GetTaskByID(id int64) (*models.Task, error) {
	query, args, err := r.sb.Select("title", "description", "user_id", "due_date", "status", "created_at").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query")
	}

	var task models.Task
	err = r.db.Get(&task, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch task")
	}

	return &task, nil
}

// GetTasksByUserID retrieves a task by user ID
func (r *TaskRepository) GetTasksByUserID(userID int64) ([]*models.Task, error) {
	query, args, err := r.sb.Select("title", "description", "user_id", "due_date", "status", "created_at").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query")
	}

	var tasks []*models.Task
	err = r.db.Select(&tasks, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch tasks")
	}

	return tasks, nil
}

// UpdateTask updates an existing task
func (r *TaskRepository) UpdateTask(task *models.Task) error {
	query, args, err := r.sb.Update("tasks").
		Set("title", task.Title).
		Set("description", task.Description).
		Set("status", task.Status).
		Set("due_date", task.DueDate).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": task.ID}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build query")
	}

	_, err = r.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute update query")
	}

	return nil
}

// DeleteTask updates a field 'deleted_at' of an existing task (soft delete)
func (r *TaskRepository) DeleteTask(id int64) error {
	query, args, err := r.sb.Update("tasks").
		Set("deleted_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build query")
	}

	_, err = r.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute update query")
	}

	return nil
}
