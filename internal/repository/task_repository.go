package repository

import (
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
)

type TaskRepository interface {
	CreateTask(task *models.Task) (int64, error)
	GetTaskByID(id int64) (*models.Task, error)
	GetTasksByUserID(userID int64) ([]*models.Task, error)
	UpdateTask(task *models.Task) error
	DeleteTask(id int64) error
}

type taskRepository struct {
	db *sqlx.DB
	sb squirrel.StatementBuilderType
}

func NewTaskRepository(db *sqlx.DB) TaskRepository {
	return &taskRepository{
		db: db,
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// CreateTask inserts a new task into the database
// default values are preset for: id, created_at, status[pending] (database-level)
// nil values are preset for: updated_at, deleted_at (database-level)
func (r *taskRepository) CreateTask(task *models.Task) (int64, error) {
	query, args, err := r.sb.Insert("tasks").
		Columns("title", "description", "user_id", "messenger_related_user_id", "start_date", "finish_date", "cron_expression").
		Values(task.Title, task.Description, task.UserID, task.MessengerRelatedUserID, task.StartDate, task.FinishDate, task.CronExpression).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query while creating new task")
	}

	var id int64
	err = r.db.QueryRow(query, args...).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert task")
	}

	return id, nil
}

// GetTaskByID retrieves a task by its ID
// Returns task entity and an error if occurred
func (r *taskRepository) GetTaskByID(id int64) (*models.Task, error) {
	query, args, err := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "start_date", "finish_date", "cron_expression", "status", "created_at").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.NotEq{"status": "done"}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query while getting task by id")
	}

	var task models.Task
	err = r.db.Get(&task, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Wrap(errs.ErrNotFound, "no task found for passed id")
		}

		return nil, errors.Wrap(err, "failed to get task by id")
	}

	return &task, nil
}

// GetTasksByUserID retrieves a task by user ID
// Returns task entities for passed user ID and an error if occurred
func (r *taskRepository) GetTasksByUserID(userID int64) ([]*models.Task, error) {
	query, args, err := r.sb.Select("id", "title", "description", "user_id", "start_date", "finish_date", "cron_expression", "status", "created_at").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.NotEq{"status": "done"}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query while getting tasks by user id")
	}

	var tasks []*models.Task
	err = r.db.Select(&tasks, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tasks by user id")
	}

	return tasks, nil
}

// UpdateTask updates task with not nil fields passed in request
// It sets the updated_at to the current time
func (r *taskRepository) UpdateTask(task *models.Task) error {
	query, args, err := r.sb.Update("tasks").
		Set("title", task.Title).
		Set("description", task.Description).
		Set("status", task.Status).
		Set("start_date", task.StartDate).
		Set("finish_date", task.FinishDate).
		Set("cron_expression", task.CronExpression).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": task.ID}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build query while updating task")
	}

	_, err = r.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute update query for task")
	}

	return nil
}

// DeleteTask soft deletes task by its id
// It sets the deleted_at timestamp to the current time
func (r *taskRepository) DeleteTask(id int64) error {
	query, args, err := r.sb.Update("tasks").
		Set("deleted_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build query while soft deleting task")
	}

	_, err = r.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute soft delete query for task")
	}

	return nil
}
