package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/rs/zerolog"
)

type TaskHistoryRepository interface {
	CreateTaskHistory(ctx context.Context, history *models.TaskHistory) error
	GetTaskHistoryByTaskID(ctx context.Context, taskID int64) ([]*models.TaskHistory, error)
	GetTaskHistoryByUserID(ctx context.Context, userID int64, limit, offset int) ([]*models.TaskHistory, error)
}

type taskHistoryRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

// TaskHistoryRow is used to scan JSONB fields as []byte before unmarshaling
type TaskHistoryRow struct {
	ID        int64     `db:"id"`
	TaskID    int64     `db:"task_id"`
	UserID    int64     `db:"user_id"`
	Action    string    `db:"action"`
	OldValue  []byte    `db:"old_value"`
	NewValue  []byte    `db:"new_value"`
	CreatedAt time.Time `db:"created_at"`
}

func NewTaskHistoryRepository(db *sqlx.DB, logger zerolog.Logger) TaskHistoryRepository {
	return &taskHistoryRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("task-history-repository"),
		logger: logger,
	}
}

// CreateTaskHistory inserts a new task history entry into the database
func (r *taskHistoryRepository) CreateTaskHistory(ctx context.Context, history *models.TaskHistory) error {
	ctx, span := r.tracer.Start(ctx, "task_history_repository.CreateTaskHistory",
		trace.WithAttributes(
			attribute.Int64("task.id", history.TaskID),
			attribute.Int64("user.id", history.UserID),
			attribute.String("action", history.Action),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task.id", history.TaskID).
		Int64("user.id", history.UserID).
		Str("action", history.Action).
		Msg("creating task history in database")

	var oldValueJSON, newValueJSON []byte
	var err error

	if history.OldValue != nil {
		oldValueJSON, err = json.Marshal(history.OldValue)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return errors.Wrap(err, "failed to marshal old_value")
		}
	}

	if history.NewValue != nil {
		newValueJSON, err = json.Marshal(history.NewValue)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return errors.Wrap(err, "failed to marshal new_value")
		}
	}

	query, args, err := r.sb.Insert("task_history").
		Columns("task_id", "user_id", "action", "old_value", "new_value", "created_at").
		Values(history.TaskID, history.UserID, history.Action, oldValueJSON, newValueJSON, time.Now().UTC()).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while creating task history")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", history.TaskID).
			Int64("user.id", history.UserID).
			Str("action", history.Action).
			Msg("failed to create task history in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to insert task history")
	}

	log.Debug().
		Int64("task.id", history.TaskID).
		Int64("user.id", history.UserID).
		Str("action", history.Action).
		Msg("task history created successfully in database")
	span.SetStatus(codes.Ok, "task history created successfully")
	return nil
}

// GetTaskHistoryByTaskID retrieves task history entries by task ID
func (r *taskHistoryRepository) GetTaskHistoryByTaskID(ctx context.Context, taskID int64) ([]*models.TaskHistory, error) {
	ctx, span := r.tracer.Start(ctx, "task_history_repository.GetTaskHistoryByTaskID",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("task.id", taskID).
		Msg("getting task history by task id from database")

	query, args, err := r.sb.Select("id", "task_id", "user_id", "action", "old_value", "new_value", "created_at").
		From("task_history").
		Where(squirrel.Eq{"task_id": taskID}).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting task history by task id")
	}

	var rows []TaskHistoryRow
	err = r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to get task history by task id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get task history by task id")
	}

	histories := make([]*models.TaskHistory, 0, len(rows))
	for _, row := range rows {
		history := &models.TaskHistory{
			ID:        row.ID,
			TaskID:    row.TaskID,
			UserID:    row.UserID,
			Action:    row.Action,
			CreatedAt: row.CreatedAt,
		}

		if len(row.OldValue) > 0 {
			var oldValue map[string]interface{}
			if err := json.Unmarshal(row.OldValue, &oldValue); err == nil {
				history.OldValue = oldValue
			}
		}

		if len(row.NewValue) > 0 {
			var newValue map[string]interface{}
			if err := json.Unmarshal(row.NewValue, &newValue); err == nil {
				history.NewValue = newValue
			}
		}

		histories = append(histories, history)
	}

	log.Debug().
		Int64("task.id", taskID).
		Int("history.count", len(histories)).
		Msg("task history retrieved successfully from database")
	span.SetAttributes(attribute.Int("history.count", len(histories)))
	span.SetStatus(codes.Ok, "task history retrieved successfully")
	return histories, nil
}

// GetTaskHistoryByUserID retrieves task history entries by user ID with pagination
func (r *taskHistoryRepository) GetTaskHistoryByUserID(ctx context.Context, userID int64, limit, offset int) ([]*models.TaskHistory, error) {
	ctx, span := r.tracer.Start(ctx, "task_history_repository.GetTaskHistoryByUserID",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
			attribute.Int("limit", limit),
			attribute.Int("offset", offset),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", userID).
		Int("limit", limit).
		Int("offset", offset).
		Msg("getting task history by user id from database")

	if limit <= 0 {
		limit = 50 // default limit
	}
	if offset < 0 {
		offset = 0
	}

	query, args, err := r.sb.Select("id", "task_id", "user_id", "action", "old_value", "new_value", "created_at").
		From("task_history").
		Where(squirrel.Eq{"user_id": userID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting task history by user id")
	}

	var rows []TaskHistoryRow
	err = r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get task history by user id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get task history by user id")
	}

	histories := make([]*models.TaskHistory, 0, len(rows))
	for _, row := range rows {
		history := &models.TaskHistory{
			ID:        row.ID,
			TaskID:    row.TaskID,
			UserID:    row.UserID,
			Action:    row.Action,
			CreatedAt: row.CreatedAt,
		}

		if len(row.OldValue) > 0 {
			var oldValue map[string]interface{}
			if err := json.Unmarshal(row.OldValue, &oldValue); err == nil {
				history.OldValue = oldValue
			}
		}

		if len(row.NewValue) > 0 {
			var newValue map[string]interface{}
			if err := json.Unmarshal(row.NewValue, &newValue); err == nil {
				history.NewValue = newValue
			}
		}

		histories = append(histories, history)
	}

	log.Debug().
		Int64("user.id", userID).
		Int("history.count", len(histories)).
		Msg("task history retrieved successfully from database")
	span.SetAttributes(attribute.Int("history.count", len(histories)))
	span.SetStatus(codes.Ok, "task history retrieved successfully")
	return histories, nil
}
