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
	GetTaskByIDWithoutStatusFilter(ctx context.Context, id int64) (*models.Task, error)
	GetTasksByUserID(ctx context.Context, userID int64) ([]*models.Task, error)
	GetTasksByUserIDWithPagination(ctx context.Context, userID int64, page, pageSize int, orderBy string, startDateFrom, startDateTo, createdAtFrom, createdAtTo *time.Time, requiresConfirmation *bool, status *string, statusNot *string, cronExpression *string, cronExpressionIsNull *bool, excludeCronWithConfirmation *bool) ([]*models.Task, int, error)
	GetChildTasksByParentID(ctx context.Context, parentID int64) ([]*models.Task, error)
	UpdateTask(ctx context.Context, task *models.Task) error
	UpdateTaskWithTx(ctx context.Context, tx *sqlx.Tx, task *models.Task) error
	DeleteTask(ctx context.Context, id int64) error
	DeleteTaskWithTx(ctx context.Context, tx *sqlx.Tx, id int64) error
	DeleteChildTasks(ctx context.Context, parentID int64) error
	DeleteChildTasksWithTx(ctx context.Context, tx *sqlx.Tx, parentID int64) error
	GetTasksNeedingRescheduling(ctx context.Context) ([]*models.Task, error)
	GetTasksWithCronNeedingRescheduling(ctx context.Context) ([]*models.Task, error)
	GetDB() *sqlx.DB
	GetAllTasks(ctx context.Context, page, pageSize int, orderBy string, status *string, statusNot *string, startDateFrom *time.Time, startDateTo *time.Time, userID *int64, cronExpression *string, cronExpressionIsNull *bool, requiresConfirmation *bool, excludeCronWithConfirmation *bool) ([]*models.Task, int, error)
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
		Columns("title", "description", "user_id", "messenger_related_user_id", "status", "parent_id", "start_date", "finish_date", "cron_expression", "rrule", "requires_confirmation").
		Values(task.Title, task.Description, task.UserID, task.MessengerRelatedUserID, task.Status, task.ParentID, task.StartDate, task.FinishDate, task.CronExpression, task.RRule, task.RequiresConfirmation).
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

	query, args, err := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "parent_id", "start_date", "finish_date", "cron_expression", "rrule", "status", "created_at", "requires_confirmation").
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

// GetTaskByIDWithoutStatusFilter retrieves a task by its ID without filtering by status
// This is useful for operations that need to check task status regardless of current state
func (r *taskRepository) GetTaskByIDWithoutStatusFilter(ctx context.Context, id int64) (*models.Task, error) {
	ctx, span := r.tracer.Start(ctx, "task_repository.GetTaskByIDWithoutStatusFilter",
		trace.WithAttributes(
			attribute.Int64("task.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task.id", id).
		Msg("getting task by id from database (without status filter)")

	query, args, err := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "parent_id", "start_date", "finish_date", "cron_expression", "rrule", "status", "created_at", "requires_confirmation").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
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
		Str("task.status", task.Status).
		Msg("task retrieved successfully from database")
	span.SetAttributes(
		attribute.Int64("user.id", task.UserID),
		attribute.String("task.status", task.Status),
	)
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

	query, args, err := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "parent_id", "start_date", "finish_date", "cron_expression", "rrule", "status", "created_at", "requires_confirmation").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.Or{
			squirrel.Eq{"cron_expression": nil},
			squirrel.NotEq{"parent_id": nil},
		}).
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

// GetTasksByUserIDWithPagination retrieves tasks by user ID with pagination and ordering
// Returns task entities, total count, and an error if occurred
func (r *taskRepository) GetTasksByUserIDWithPagination(ctx context.Context, userID int64, page, pageSize int, orderBy string, startDateFrom, startDateTo, createdAtFrom, createdAtTo *time.Time, requiresConfirmation *bool, status *string, statusNot *string, cronExpression *string, cronExpressionIsNull *bool, excludeCronWithConfirmation *bool) ([]*models.Task, int, error) {
	ctx, span := r.tracer.Start(ctx, "task_repository.GetTasksByUserIDWithPagination",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
			attribute.Int("page", page),
			attribute.Int("page_size", pageSize),
			attribute.String("order_by", orderBy),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", userID).
		Int("page", page).
		Int("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting tasks by user id from database with pagination")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	offset := (page - 1) * pageSize

	// Build count query with filters
	countBuilder := r.sb.Select("COUNT(*)").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"user_id": userID})

	// Apply filters
	if status != nil && *status != "" {
		countBuilder = countBuilder.Where(squirrel.Eq{"status": *status})
		span.SetAttributes(attribute.String("filter.status", *status))
	}
	if statusNot != nil && *statusNot != "" {
		countBuilder = countBuilder.Where(squirrel.NotEq{"status": *statusNot})
		span.SetAttributes(attribute.String("filter.status_not", *statusNot))
	}
	if startDateFrom != nil {
		countBuilder = countBuilder.Where(squirrel.GtOrEq{"start_date": *startDateFrom})
		span.SetAttributes(attribute.String("filter.start_date_from", startDateFrom.Format(time.RFC3339)))
	}
	if startDateTo != nil {
		countBuilder = countBuilder.Where(squirrel.LtOrEq{"start_date": *startDateTo})
		span.SetAttributes(attribute.String("filter.start_date_to", startDateTo.Format(time.RFC3339)))
	}
	if createdAtFrom != nil {
		countBuilder = countBuilder.Where(squirrel.GtOrEq{"created_at": *createdAtFrom})
		span.SetAttributes(attribute.String("filter.created_at_from", createdAtFrom.Format(time.RFC3339)))
	}
	if createdAtTo != nil {
		countBuilder = countBuilder.Where(squirrel.LtOrEq{"created_at": *createdAtTo})
		span.SetAttributes(attribute.String("filter.created_at_to", createdAtTo.Format(time.RFC3339)))
	}
	if cronExpression != nil && *cronExpression != "" {
		countBuilder = countBuilder.Where(squirrel.Eq{"cron_expression": *cronExpression})
		span.SetAttributes(attribute.String("filter.cron_expression", *cronExpression))
	}
	if cronExpressionIsNull != nil {
		if *cronExpressionIsNull {
			countBuilder = countBuilder.Where(squirrel.Eq{"cron_expression": nil})
			span.SetAttributes(attribute.Bool("filter.cron_expression_is_null", true))
		} else {
			countBuilder = countBuilder.Where(squirrel.NotEq{"cron_expression": nil})
			span.SetAttributes(attribute.Bool("filter.cron_expression_is_null", false))
		}
	}
	if requiresConfirmation != nil {
		countBuilder = countBuilder.Where(squirrel.Eq{"requires_confirmation": *requiresConfirmation})
		span.SetAttributes(attribute.Bool("filter.requires_confirmation", *requiresConfirmation))
	}
	// Exclude tasks where cron_expression IS NOT NULL AND requires_confirmation == True at the same time
	// This implements: NOT (cron_expression IS NOT NULL AND requires_confirmation == True)
	// Using raw SQL expression to properly exclude: (cron_expression IS NULL OR requires_confirmation = false)
	if excludeCronWithConfirmation != nil && *excludeCronWithConfirmation {
		countBuilder = countBuilder.Where(squirrel.Expr("(cron_expression IS NULL OR requires_confirmation = false)"))
		span.SetAttributes(attribute.Bool("filter.exclude_cron_with_confirmation", true))
	}

	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to build count query")
	}

	var totalCount int
	err = r.db.GetContext(ctx, &totalCount, countQuery, countArgs...)
	if err != nil {
		log.Debug().Err(err).Msg("failed to get total count of tasks by user id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get total count")
	}

	// Build data query with filters
	dataBuilder := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "parent_id", "start_date", "finish_date", "cron_expression", "rrule", "status", "created_at", "requires_confirmation").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"user_id": userID})

	// Apply filters
	if status != nil && *status != "" {
		dataBuilder = dataBuilder.Where(squirrel.Eq{"status": *status})
	}
	if statusNot != nil && *statusNot != "" {
		dataBuilder = dataBuilder.Where(squirrel.NotEq{"status": *statusNot})
	}
	if startDateFrom != nil {
		dataBuilder = dataBuilder.Where(squirrel.GtOrEq{"start_date": *startDateFrom})
	}
	if startDateTo != nil {
		dataBuilder = dataBuilder.Where(squirrel.LtOrEq{"start_date": *startDateTo})
	}
	if createdAtFrom != nil {
		dataBuilder = dataBuilder.Where(squirrel.GtOrEq{"created_at": *createdAtFrom})
	}
	if createdAtTo != nil {
		dataBuilder = dataBuilder.Where(squirrel.LtOrEq{"created_at": *createdAtTo})
	}
	if cronExpression != nil && *cronExpression != "" {
		dataBuilder = dataBuilder.Where(squirrel.Eq{"cron_expression": *cronExpression})
	}
	if cronExpressionIsNull != nil {
		if *cronExpressionIsNull {
			dataBuilder = dataBuilder.Where(squirrel.Eq{"cron_expression": nil})
		} else {
			dataBuilder = dataBuilder.Where(squirrel.NotEq{"cron_expression": nil})
		}
	}
	if requiresConfirmation != nil {
		dataBuilder = dataBuilder.Where(squirrel.Eq{"requires_confirmation": *requiresConfirmation})
	}
	// Exclude tasks where cron_expression IS NOT NULL AND requires_confirmation == True at the same time
	// This implements: NOT (cron_expression IS NOT NULL AND requires_confirmation == True)
	// Using raw SQL expression to properly exclude: (cron_expression IS NULL OR requires_confirmation = false)
	if excludeCronWithConfirmation != nil && *excludeCronWithConfirmation {
		dataBuilder = dataBuilder.Where(squirrel.Expr("(cron_expression IS NULL OR requires_confirmation = false)"))
	}

	query, args, err := dataBuilder.
		OrderBy(orderBy).
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to build query")
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
		return nil, 0, errors.Wrap(err, "failed to get tasks by user id")
	}

	log.Debug().
		Int64("user.id", userID).
		Int("tasks.count", len(tasks)).
		Int("total_count", totalCount).
		Msg("tasks retrieved successfully from database")
	span.SetAttributes(attribute.Int("tasks.count", len(tasks)))
	span.SetAttributes(attribute.Int("total_count", totalCount))
	span.SetStatus(codes.Ok, "tasks retrieved successfully")
	return tasks, totalCount, nil
}

// GetChildTasksByParentID retrieves all child tasks by parent ID
// Returns child task entities for passed parent ID and an error if occurred
func (r *taskRepository) GetChildTasksByParentID(ctx context.Context, parentID int64) ([]*models.Task, error) {
	ctx, span := r.tracer.Start(ctx, "task_repository.GetChildTasksByParentID",
		trace.WithAttributes(
			attribute.Int64("parent.id", parentID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("parent.id", parentID).
		Msg("getting child tasks by parent id from database")

	query, args, err := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "parent_id", "start_date", "finish_date", "cron_expression", "rrule", "status", "created_at", "requires_confirmation").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"parent_id": parentID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting child tasks by parent id")
	}

	var tasks []*models.Task
	err = r.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("parent.id", parentID).
			Msg("failed to get child tasks by parent id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get child tasks by parent id")
	}

	log.Debug().
		Int64("parent.id", parentID).
		Int("tasks.count", len(tasks)).
		Msg("child tasks retrieved successfully from database")
	span.SetAttributes(attribute.Int("tasks.count", len(tasks)))
	span.SetStatus(codes.Ok, "child tasks retrieved successfully")
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
		Set("rrule", task.RRule).
		Set("requires_confirmation", task.RequiresConfirmation).
		Set("parent_id", task.ParentID).
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

// UpdateTaskWithTx updates task within a transaction
// It sets the updated_at to the current time
func (r *taskRepository) UpdateTaskWithTx(ctx context.Context, tx *sqlx.Tx, task *models.Task) error {
	ctx, span := r.tracer.Start(ctx, "task_repository.UpdateTaskWithTx",
		trace.WithAttributes(
			attribute.Int64("task.id", task.ID),
			attribute.String("task.status", task.Status),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task.id", task.ID).
		Str("task.status", task.Status).
		Msg("updating task in database within transaction")

	query, args, err := r.sb.Update("tasks").
		Set("title", task.Title).
		Set("description", task.Description).
		Set("status", task.Status).
		Set("start_date", task.StartDate).
		Set("finish_date", task.FinishDate).
		Set("cron_expression", task.CronExpression).
		Set("rrule", task.RRule).
		Set("requires_confirmation", task.RequiresConfirmation).
		Set("parent_id", task.ParentID).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": task.ID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while updating task")
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", task.ID).
			Msg("failed to update task in database within transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute update query for task")
	}

	log.Debug().
		Int64("task.id", task.ID).
		Msg("task updated successfully in database within transaction")
	span.SetStatus(codes.Ok, "task updated successfully")
	return nil
}

// GetDB returns the underlying database connection
// This is needed for transaction management at the service level
func (r *taskRepository) GetDB() *sqlx.DB {
	return r.db
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

// DeleteTaskWithTx soft deletes task by its id within a transaction
// It sets the deleted_at timestamp to the current time
func (r *taskRepository) DeleteTaskWithTx(ctx context.Context, tx *sqlx.Tx, id int64) error {
	ctx, span := r.tracer.Start(ctx, "task_repository.DeleteTaskWithTx",
		trace.WithAttributes(
			attribute.Int64("task.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task.id", id).
		Msg("deleting task from database within transaction")

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

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", id).
			Msg("failed to delete task from database within transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute soft delete query for task")
	}

	log.Debug().
		Int64("task.id", id).
		Msg("task deleted successfully from database within transaction")
	span.SetStatus(codes.Ok, "task deleted successfully")
	return nil
}

// DeleteChildTasks soft deletes all child tasks by parent id
// It sets the deleted_at timestamp to the current time for all child tasks
func (r *taskRepository) DeleteChildTasks(ctx context.Context, parentID int64) error {
	ctx, span := r.tracer.Start(ctx, "task_repository.DeleteChildTasks",
		trace.WithAttributes(
			attribute.Int64("parent.id", parentID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("parent.id", parentID).
		Msg("deleting child tasks from database")

	query, args, err := r.sb.Update("tasks").
		Set("deleted_at", time.Now().UTC()).
		Set("status", "deleted").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"parent_id": parentID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while soft deleting child tasks")
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("parent.id", parentID).
			Msg("failed to delete child tasks from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute soft delete query for child tasks")
	}

	rowsAffected, _ := result.RowsAffected()
	log.Debug().
		Int64("parent.id", parentID).
		Int64("rows_affected", rowsAffected).
		Msg("child tasks deleted successfully from database")
	span.SetAttributes(attribute.Int64("rows_affected", rowsAffected))
	span.SetStatus(codes.Ok, "child tasks deleted successfully")
	return nil
}

// DeleteChildTasksWithTx soft deletes all child tasks by parent id within a transaction
// It sets the deleted_at timestamp to the current time for all child tasks
func (r *taskRepository) DeleteChildTasksWithTx(ctx context.Context, tx *sqlx.Tx, parentID int64) error {
	ctx, span := r.tracer.Start(ctx, "task_repository.DeleteChildTasksWithTx",
		trace.WithAttributes(
			attribute.Int64("parent.id", parentID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("parent.id", parentID).
		Msg("deleting child tasks from database within transaction")

	query, args, err := r.sb.Update("tasks").
		Set("deleted_at", time.Now().UTC()).
		Set("status", "deleted").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"parent_id": parentID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while soft deleting child tasks")
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("parent.id", parentID).
			Msg("failed to delete child tasks from database within transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute soft delete query for child tasks")
	}

	rowsAffected, _ := result.RowsAffected()
	log.Debug().
		Int64("parent.id", parentID).
		Int64("rows_affected", rowsAffected).
		Msg("child tasks deleted successfully from database within transaction")
	span.SetAttributes(attribute.Int64("rows_affected", rowsAffected))
	span.SetStatus(codes.Ok, "child tasks deleted successfully")
	return nil
}

// GetTasksNeedingRescheduling retrieves tasks that need to be rescheduled:
// - no cron expression (cron_expression IS NULL)
// - status is 'scheduled'
// - start_date has passed (start_date < NOW())
func (r *taskRepository) GetTasksNeedingRescheduling(ctx context.Context) ([]*models.Task, error) {
	ctx, span := r.tracer.Start(ctx, "task_repository.GetTasksNeedingRescheduling")
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Msg("getting tasks needing rescheduling from database")

	query, args, err := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "parent_id", "start_date", "finish_date", "cron_expression", "rrule", "status", "created_at", "requires_confirmation").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Or{
			squirrel.Eq{"status": string(models.TaskStatusScheduled)},
			squirrel.Eq{"status": string(models.TaskStatusRescheduled)},
			squirrel.Eq{"status": string(models.TaskStatusPostponed)},
		}).
		Where(squirrel.Or{
			squirrel.Eq{"cron_expression": nil},
			squirrel.NotEq{"parent_id": nil},
		}).
		Where(squirrel.Lt{"start_date": time.Now().UTC()}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting tasks needing rescheduling")
	}

	var tasks []*models.Task
	err = r.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get tasks needing rescheduling from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get tasks needing rescheduling")
	}

	log.Debug().
		Int("tasks.count", len(tasks)).
		Msg("tasks needing rescheduling retrieved successfully from database")
	span.SetAttributes(attribute.Int("tasks.count", len(tasks)))
	span.SetStatus(codes.Ok, "tasks needing rescheduling retrieved successfully")
	return tasks, nil
}

// GetTasksWithCronNeedingRescheduling retrieves tasks that need to be rescheduled:
// - have cron expression (cron_expression IS NOT NULL)
// - requires_confirmation = false
// - status is 'scheduled'
// - start_date has passed (start_date < NOW())
func (r *taskRepository) GetTasksWithCronNeedingRescheduling(ctx context.Context) ([]*models.Task, error) {
	ctx, span := r.tracer.Start(ctx, "task_repository.GetTasksWithCronNeedingRescheduling")
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Msg("getting tasks with cron needing rescheduling from database")

	query, args, err := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "parent_id", "start_date", "finish_date", "cron_expression", "rrule", "status", "created_at", "requires_confirmation").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"status": string(models.TaskStatusScheduled)}).
		Where(squirrel.NotEq{"cron_expression": nil}).
		Where(squirrel.Eq{"requires_confirmation": false}).
		Where(squirrel.Lt{"start_date": time.Now().UTC()}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting tasks with cron needing rescheduling")
	}

	var tasks []*models.Task
	err = r.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get tasks with cron needing rescheduling from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get tasks with cron needing rescheduling")
	}

	log.Debug().
		Int("tasks.count", len(tasks)).
		Msg("tasks with cron needing rescheduling retrieved successfully from database")
	span.SetAttributes(attribute.Int("tasks.count", len(tasks)))
	span.SetStatus(codes.Ok, "tasks with cron needing rescheduling retrieved successfully")
	return tasks, nil
}

// GetAllTasks retrieves all tasks with pagination, ordering, and filtering
func (r *taskRepository) GetAllTasks(ctx context.Context, page, pageSize int, orderBy string, status *string, statusNot *string, startDateFrom *time.Time, startDateTo *time.Time, userID *int64, cronExpression *string, cronExpressionIsNull *bool, requiresConfirmation *bool, excludeCronWithConfirmation *bool) ([]*models.Task, int, error) {
	ctx, span := r.tracer.Start(ctx, "task_repository.GetAllTasks",
		trace.WithAttributes(
			attribute.Int("page", page),
			attribute.Int("page_size", pageSize),
			attribute.String("order_by", orderBy),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int("page", page).
		Int("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting all tasks from database")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	offset := (page - 1) * pageSize

	// Build count query with filters
	countBuilder := r.sb.Select("COUNT(*)").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil})

	if status != nil && *status != "" {
		countBuilder = countBuilder.Where(squirrel.Eq{"status": *status})
		span.SetAttributes(attribute.String("filter.status", *status))
	}
	if statusNot != nil && *statusNot != "" {
		countBuilder = countBuilder.Where(squirrel.NotEq{"status": *statusNot})
		span.SetAttributes(attribute.String("filter.status_not", *statusNot))
	}
	if startDateFrom != nil {
		countBuilder = countBuilder.Where(squirrel.GtOrEq{"start_date": *startDateFrom})
		span.SetAttributes(attribute.String("filter.start_date_from", startDateFrom.Format(time.RFC3339)))
	}
	if startDateTo != nil {
		countBuilder = countBuilder.Where(squirrel.LtOrEq{"start_date": *startDateTo})
		span.SetAttributes(attribute.String("filter.start_date_to", startDateTo.Format(time.RFC3339)))
	}
	if userID != nil {
		countBuilder = countBuilder.Where(squirrel.Eq{"user_id": *userID})
		span.SetAttributes(attribute.Int64("filter.user_id", *userID))
	}
	if cronExpression != nil && *cronExpression != "" {
		countBuilder = countBuilder.Where(squirrel.Eq{"cron_expression": *cronExpression})
		span.SetAttributes(attribute.String("filter.cron_expression", *cronExpression))
	}
	if cronExpressionIsNull != nil {
		if *cronExpressionIsNull {
			countBuilder = countBuilder.Where(squirrel.Eq{"cron_expression": nil})
			span.SetAttributes(attribute.Bool("filter.cron_expression_is_null", true))
		} else {
			countBuilder = countBuilder.Where(squirrel.NotEq{"cron_expression": nil})
			span.SetAttributes(attribute.Bool("filter.cron_expression_is_null", false))
		}
	}
	if requiresConfirmation != nil {
		countBuilder = countBuilder.Where(squirrel.Eq{"requires_confirmation": *requiresConfirmation})
		span.SetAttributes(attribute.Bool("filter.requires_confirmation", *requiresConfirmation))
	}
	// Exclude tasks where cron_expression IS NOT NULL AND requires_confirmation == True at the same time
	// This implements: NOT (cron_expression IS NOT NULL AND requires_confirmation == True)
	// Using raw SQL expression to properly exclude: (cron_expression IS NULL OR requires_confirmation = false)
	if excludeCronWithConfirmation != nil && *excludeCronWithConfirmation {
		countBuilder = countBuilder.Where(squirrel.Expr("(cron_expression IS NULL OR requires_confirmation = false)"))
		span.SetAttributes(attribute.Bool("filter.exclude_cron_with_confirmation", true))
	}

	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to build count query")
	}

	var totalCount int
	err = r.db.GetContext(ctx, &totalCount, countQuery, countArgs...)
	if err != nil {
		log.Debug().Err(err).Msg("failed to get total count of tasks")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get total count")
	}

	// Build data query with filters
	dataBuilder := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "parent_id", "start_date", "finish_date", "cron_expression", "rrule", "status", "created_at", "requires_confirmation").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil})

	if status != nil && *status != "" {
		dataBuilder = dataBuilder.Where(squirrel.Eq{"status": *status})
	}
	if statusNot != nil && *statusNot != "" {
		dataBuilder = dataBuilder.Where(squirrel.NotEq{"status": *statusNot})
	}
	if startDateFrom != nil {
		dataBuilder = dataBuilder.Where(squirrel.GtOrEq{"start_date": *startDateFrom})
	}
	if startDateTo != nil {
		dataBuilder = dataBuilder.Where(squirrel.LtOrEq{"start_date": *startDateTo})
	}
	if userID != nil {
		dataBuilder = dataBuilder.Where(squirrel.Eq{"user_id": *userID})
	}
	if cronExpression != nil && *cronExpression != "" {
		dataBuilder = dataBuilder.Where(squirrel.Eq{"cron_expression": *cronExpression})
	}
	if cronExpressionIsNull != nil {
		if *cronExpressionIsNull {
			dataBuilder = dataBuilder.Where(squirrel.Eq{"cron_expression": nil})
		} else {
			dataBuilder = dataBuilder.Where(squirrel.NotEq{"cron_expression": nil})
		}
	}
	if requiresConfirmation != nil {
		dataBuilder = dataBuilder.Where(squirrel.Eq{"requires_confirmation": *requiresConfirmation})
	}
	// Exclude tasks where cron_expression IS NOT NULL AND requires_confirmation == True
	// This implements: NOT (cron_expression IS NOT NULL AND requires_confirmation == True)
	// Using raw SQL expression to properly exclude: (cron_expression IS NULL OR requires_confirmation = false)
	if excludeCronWithConfirmation != nil && *excludeCronWithConfirmation {
		dataBuilder = dataBuilder.Where(squirrel.Expr("(cron_expression IS NULL OR requires_confirmation = false)"))
	}

	query, args, err := dataBuilder.
		OrderBy(orderBy).
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to build query")
	}

	var tasks []*models.Task
	err = r.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		log.Debug().Err(err).Msg("failed to get tasks from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get tasks")
	}

	log.Debug().
		Int("tasks.count", len(tasks)).
		Int("total_count", totalCount).
		Msg("tasks retrieved successfully from database")
	span.SetAttributes(
		attribute.Int("tasks.count", len(tasks)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "tasks retrieved successfully")
	return tasks, totalCount, nil
}
