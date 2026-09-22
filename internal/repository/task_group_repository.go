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

type TaskGroupRepository interface {
	CreateTaskGroup(ctx context.Context, group *models.TaskGroup) (int64, error)
	GetTaskGroupByID(ctx context.Context, id int64) (*models.TaskGroup, error)
	GetAllTaskGroups(ctx context.Context, page, pageSize int, orderBy string, userID *int64) ([]*models.TaskGroup, int, error)
	UpdateTaskGroup(ctx context.Context, group *models.TaskGroup) error
	DeleteTaskGroup(ctx context.Context, id int64) error
}

type taskGroupRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewTaskGroupRepository(db *sqlx.DB, logger zerolog.Logger) TaskGroupRepository {
	return &taskGroupRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("task-group-repository"),
		logger: logger,
	}
}

func (r *taskGroupRepository) CreateTaskGroup(ctx context.Context, group *models.TaskGroup) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "task_group_repository.CreateTaskGroup",
		trace.WithAttributes(
			attribute.Int64("user.id", group.UserID),
			attribute.String("task_group.name", group.Name),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", group.UserID).
		Str("task_group.name", group.Name).
		Msg("creating task group in database")

	query, args, err := r.sb.Insert("task_groups").
		Columns("user_id", "name").
		Values(group.UserID, group.Name).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while creating new task group")
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to insert task group")
	}

	span.SetAttributes(attribute.Int64("task_group.id", id))
	span.SetStatus(codes.Ok, "task group created successfully")
	return id, nil
}

func (r *taskGroupRepository) GetTaskGroupByID(ctx context.Context, id int64) (*models.TaskGroup, error) {
	ctx, span := r.tracer.Start(ctx, "task_group_repository.GetTaskGroupByID",
		trace.WithAttributes(
			attribute.Int64("task_group.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task_group.id", id).
		Msg("getting task group by id from database")

	query, args, err := r.sb.Select("id", "user_id", "name", "created_at", "updated_at").
		From("task_groups").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting task group by id")
	}

	var group models.TaskGroup
	err = r.db.GetContext(ctx, &group, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no task group found for passed id")
			log.Debug().
				Err(err).
				Int64("task_group.id", id).
				Msg("task group not found")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		log.Debug().
			Err(err).
			Int64("task_group.id", id).
			Msg("failed to get task group by id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get task group by id")
	}

	log.Debug().
		Int64("task_group.id", id).
		Int64("user.id", group.UserID).
		Msg("task group retrieved successfully from database")
	span.SetAttributes(attribute.Int64("user.id", group.UserID))
	span.SetStatus(codes.Ok, "task group retrieved successfully")
	return &group, nil
}

func (r *taskGroupRepository) GetAllTaskGroups(ctx context.Context, page, pageSize int, orderBy string, userID *int64) ([]*models.TaskGroup, int, error) {
	ctx, span := r.tracer.Start(ctx, "task_group_repository.GetAllTaskGroups",
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
		Msg("getting all task groups from database")

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

	countBuilder := r.sb.Select("COUNT(*)").
		From("task_groups").
		Where(squirrel.Eq{"deleted_at": nil})

	if userID != nil {
		countBuilder = countBuilder.Where(squirrel.Eq{"user_id": *userID})
		span.SetAttributes(attribute.Int64("user.id", *userID))
	}

	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to build count query while getting all task groups")
	}

	var totalCount int
	err = r.db.GetContext(ctx, &totalCount, countQuery, countArgs...)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get total count of task groups from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get total count of task groups")
	}

	dataBuilder := r.sb.Select("id", "user_id", "name", "created_at", "updated_at").
		From("task_groups").
		Where(squirrel.Eq{"deleted_at": nil})

	if userID != nil {
		dataBuilder = dataBuilder.Where(squirrel.Eq{"user_id": *userID})
	}

	query, args, err := dataBuilder.
		OrderBy(orderBy).
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to build query while getting all task groups")
	}

	var groups []*models.TaskGroup
	err = r.db.SelectContext(ctx, &groups, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all task groups from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get all task groups")
	}

	log.Debug().
		Int("task_groups.count", len(groups)).
		Int("total_count", totalCount).
		Msg("task groups retrieved successfully from database")
	span.SetAttributes(
		attribute.Int("task_groups.count", len(groups)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "task groups retrieved successfully")
	return groups, totalCount, nil
}

func (r *taskGroupRepository) UpdateTaskGroup(ctx context.Context, group *models.TaskGroup) error {
	ctx, span := r.tracer.Start(ctx, "task_group_repository.UpdateTaskGroup",
		trace.WithAttributes(
			attribute.Int64("task_group.id", group.ID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task_group.id", group.ID).
		Msg("updating task group in database")

	query, args, err := r.sb.Update("task_groups").
		Set("name", group.Name).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": group.ID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while updating task group")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task_group.id", group.ID).
			Msg("failed to update task group in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute update query for task group")
	}

	log.Debug().
		Int64("task_group.id", group.ID).
		Msg("task group updated successfully in database")
	span.SetStatus(codes.Ok, "task group updated successfully")
	return nil
}

func (r *taskGroupRepository) DeleteTaskGroup(ctx context.Context, id int64) error {
	ctx, span := r.tracer.Start(ctx, "task_group_repository.DeleteTaskGroup",
		trace.WithAttributes(
			attribute.Int64("task_group.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task_group.id", id).
		Msg("deleting task group in database")

	query, args, err := r.sb.Update("task_groups").
		Set("deleted_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while deleting task group")
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task_group.id", id).
			Msg("failed to delete task group in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute delete query for task group")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task_group.id", id).
			Msg("failed to get rows affected after deleting task group")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to get rows affected after deleting task group")
	}

	if rowsAffected == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no task group found for passed id")
		log.Debug().
			Err(err).
			Int64("task_group.id", id).
			Msg("task group not found for deletion")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	log.Debug().
		Int64("task_group.id", id).
		Msg("task group deleted successfully from database")
	span.SetStatus(codes.Ok, "task group deleted successfully")
	return nil
}
