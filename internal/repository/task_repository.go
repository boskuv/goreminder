package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/rs/zerolog"
)

type TaskRepository interface {
	CreateTask(ctx context.Context, task *models.Task) (int64, error)
	GetTaskByID(ctx context.Context, id int64) (*models.Task, error)
	GetTasksByUserID(ctx context.Context, userID int64) ([]*models.Task, error)
	UpdateTask(ctx context.Context, task *models.Task) error
	DeleteTask(ctx context.Context, id int64) error
}

type taskRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewTaskRepository(db *sqlx.DB, logger zerolog.Logger) TaskRepository {
	return &taskRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("task-repository"),
		logger: logger,
	}
}

// CreateTask inserts a new task into the database
// default values are preset for: id, created_at, status[pending] (database-level)
// nil values are preset for: updated_at, deleted_at (database-level)
func (r *taskRepository) CreateTask(ctx context.Context, task *models.Task) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "task_repository.CreateTask",
		trace.WithAttributes(
			attribute.Int64("user.id", task.UserID),
			attribute.String("task.title", task.Title),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", task.UserID).
		Str("task.title", task.Title).
		Msg("creating task in database")

	query, args, err := r.sb.Insert("tasks").
		Columns("title", "description", "user_id", "messenger_related_user_id", "start_date", "finish_date", "cron_expression").
		Values(task.Title, task.Description, task.UserID, task.MessengerRelatedUserID, task.StartDate, task.FinishDate, task.CronExpression).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while creating new task")
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to insert task")
	}

	span.SetAttributes(attribute.Int64("task.id", id))
	span.SetStatus(codes.Ok, "task created successfully")
	return id, nil
}

// GetTaskByID retrieves a task by its ID
// Returns task entity and an error if occurred
func (r *taskRepository) GetTaskByID(ctx context.Context, id int64) (*models.Task, error) {
	ctx, span := r.tracer.Start(ctx, "task_repository.GetTaskByID",
		trace.WithAttributes(
			attribute.Int64("task.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task.id", id).
		Msg("getting task by id from database")

	query, args, err := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "start_date", "finish_date", "cron_expression", "status", "created_at").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.NotEq{"status": "done"}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting task by id")
	}

	var task models.Task
	err = r.db.GetContext(ctx, &task, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no task found for passed id")
			log.Debug().
				Err(err).
				Int64("task.id", id).
				Msg("task not found")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		log.Debug().
			Err(err).
			Int64("task.id", id).
			Msg("failed to get task by id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get task by id")
	}

	log.Debug().
		Int64("task.id", id).
		Int64("user.id", task.UserID).
		Msg("task retrieved successfully from database")
	span.SetAttributes(attribute.Int64("user.id", task.UserID))
	span.SetStatus(codes.Ok, "task retrieved successfully")
	return &task, nil
}

// GetTasksByUserID retrieves a task by user ID
// Returns task entities for passed user ID and an error if occurred
func (r *taskRepository) GetTasksByUserID(ctx context.Context, userID int64) ([]*models.Task, error) {
	ctx, span := r.tracer.Start(ctx, "task_repository.GetTasksByUserID",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("getting tasks by user id from database")

	query, args, err := r.sb.Select("id", "title", "description", "user_id", "start_date", "finish_date", "cron_expression", "status", "created_at").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.NotEq{"status": "done"}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting tasks by user id")
	}

	var tasks []*models.Task
	err = r.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get tasks by user id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get tasks by user id")
	}

	log.Debug().
		Int64("user.id", userID).
		Int("tasks.count", len(tasks)).
		Msg("tasks retrieved successfully from database")
	span.SetAttributes(attribute.Int("tasks.count", len(tasks)))
	span.SetStatus(codes.Ok, "tasks retrieved successfully")
	return tasks, nil
}

// UpdateTask updates task with not nil fields passed in request
// It sets the updated_at to the current time
func (r *taskRepository) UpdateTask(ctx context.Context, task *models.Task) error {
	ctx, span := r.tracer.Start(ctx, "task_repository.UpdateTask",
		trace.WithAttributes(
			attribute.Int64("task.id", task.ID),
			attribute.String("task.status", task.Status),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task.id", task.ID).
		Str("task.status", task.Status).
		Msg("updating task in database")

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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while updating task")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", task.ID).
			Msg("failed to update task in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute update query for task")
	}

	log.Debug().
		Int64("task.id", task.ID).
		Msg("task updated successfully in database")
	span.SetStatus(codes.Ok, "task updated successfully")
	return nil
}

// DeleteTask soft deletes task by its id
// It sets the deleted_at timestamp to the current time
func (r *taskRepository) DeleteTask(ctx context.Context, id int64) error {
	ctx, span := r.tracer.Start(ctx, "task_repository.DeleteTask",
		trace.WithAttributes(
			attribute.Int64("task.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task.id", id).
		Msg("deleting task from database")

	query, args, err := r.sb.Update("tasks").
		Set("deleted_at", time.Now().UTC()).
		Set("status", "deleted").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while soft deleting task")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", id).
			Msg("failed to delete task from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute soft delete query for task")
	}

	log.Debug().
		Int64("task.id", id).
		Msg("task deleted successfully from database")
	span.SetStatus(codes.Ok, "task deleted successfully")
	return nil
}
