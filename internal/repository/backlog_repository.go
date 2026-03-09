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

type BacklogRepository interface {
	CreateBacklog(ctx context.Context, backlog *models.Backlog) (int64, error)
	GetBacklogByID(ctx context.Context, id int64) (*models.Backlog, error)
	GetAllBacklogs(ctx context.Context, page, pageSize int, orderBy string, userID *int64, completed *bool) ([]*models.Backlog, int, error)
	UpdateBacklog(ctx context.Context, backlog *models.Backlog) error
	DeleteBacklog(ctx context.Context, id int64) error
	GetCompletedBacklogsCount(ctx context.Context, userID int64, startDate, endDate time.Time) (int, error)
}

type backlogRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewBacklogRepository(db *sqlx.DB, logger zerolog.Logger) BacklogRepository {
	return &backlogRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("backlog-repository"),
		logger: logger,
	}
}

// CreateBacklog inserts a new backlog item into the database
func (r *backlogRepository) CreateBacklog(ctx context.Context, backlog *models.Backlog) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "backlog_repository.CreateBacklog",
		trace.WithAttributes(
			attribute.Int64("user.id", backlog.UserID),
			attribute.String("backlog.title", backlog.Title),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", backlog.UserID).
		Str("backlog.title", backlog.Title).
		Msg("creating backlog in database")

	query, args, err := r.sb.Insert("backlogs").
		Columns("title", "description", "user_id", "messenger_related_user_id").
		Values(backlog.Title, backlog.Description, backlog.UserID, backlog.MessengerRelatedUserID).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while creating new backlog")
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to insert backlog")
	}

	span.SetAttributes(attribute.Int64("backlog.id", id))
	span.SetStatus(codes.Ok, "backlog created successfully")
	return id, nil
}

// GetBacklogByID retrieves a backlog item by its ID
func (r *backlogRepository) GetBacklogByID(ctx context.Context, id int64) (*models.Backlog, error) {
	ctx, span := r.tracer.Start(ctx, "backlog_repository.GetBacklogByID",
		trace.WithAttributes(
			attribute.Int64("backlog.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("backlog.id", id).
		Msg("getting backlog by id from database")

	query, args, err := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "created_at", "updated_at", "completed_at").
		From("backlogs").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting backlog by id")
	}

	var backlog models.Backlog
	err = r.db.GetContext(ctx, &backlog, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no backlog found for passed id")
			log.Debug().
				Err(err).
				Int64("backlog.id", id).
				Msg("backlog not found")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		log.Debug().
			Err(err).
			Int64("backlog.id", id).
			Msg("failed to get backlog by id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get backlog by id")
	}

	log.Debug().
		Int64("backlog.id", id).
		Int64("user.id", backlog.UserID).
		Msg("backlog retrieved successfully from database")
	span.SetAttributes(attribute.Int64("user.id", backlog.UserID))
	span.SetStatus(codes.Ok, "backlog retrieved successfully")
	return &backlog, nil
}

// GetAllBacklogs retrieves all backlog items with pagination, ordering, and filtering
func (r *backlogRepository) GetAllBacklogs(ctx context.Context, page, pageSize int, orderBy string, userID *int64, completed *bool) ([]*models.Backlog, int, error) {
	ctx, span := r.tracer.Start(ctx, "backlog_repository.GetAllBacklogs",
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
		Msg("getting all backlogs from database")

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

	// Build count query
	countBuilder := r.sb.Select("COUNT(*)").
		From("backlogs").
		Where(squirrel.Eq{"deleted_at": nil})

	if userID != nil {
		countBuilder = countBuilder.Where(squirrel.Eq{"user_id": *userID})
		span.SetAttributes(attribute.Int64("user.id", *userID))
	}

	if completed != nil {
		if *completed {
			countBuilder = countBuilder.Where(squirrel.NotEq{"completed_at": nil})
		} else {
			countBuilder = countBuilder.Where(squirrel.Eq{"completed_at": nil})
		}
		span.SetAttributes(attribute.Bool("completed", *completed))
	}

	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to build count query while getting all backlogs")
	}

	var totalCount int
	err = r.db.GetContext(ctx, &totalCount, countQuery, countArgs...)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get total count of backlogs from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get total count of backlogs")
	}

	// Build data query
	dataBuilder := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "created_at", "updated_at", "completed_at").
		From("backlogs").
		Where(squirrel.Eq{"deleted_at": nil})

	if userID != nil {
		dataBuilder = dataBuilder.Where(squirrel.Eq{"user_id": *userID})
	}

	if completed != nil {
		if *completed {
			dataBuilder = dataBuilder.Where(squirrel.NotEq{"completed_at": nil})
		} else {
			dataBuilder = dataBuilder.Where(squirrel.Eq{"completed_at": nil})
		}
	}

	query, args, err := dataBuilder.
		OrderBy(orderBy).
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to build query while getting all backlogs")
	}

	var backlogs []*models.Backlog
	err = r.db.SelectContext(ctx, &backlogs, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all backlogs from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get all backlogs")
	}

	log.Debug().
		Int("backlogs.count", len(backlogs)).
		Int("total_count", totalCount).
		Msg("backlogs retrieved successfully from database")
	span.SetAttributes(
		attribute.Int("backlogs.count", len(backlogs)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "backlogs retrieved successfully")
	return backlogs, totalCount, nil
}

// UpdateBacklog updates backlog item with not nil fields passed in request
// It sets the updated_at to the current time
func (r *backlogRepository) UpdateBacklog(ctx context.Context, backlog *models.Backlog) error {
	ctx, span := r.tracer.Start(ctx, "backlog_repository.UpdateBacklog",
		trace.WithAttributes(
			attribute.Int64("backlog.id", backlog.ID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("backlog.id", backlog.ID).
		Msg("updating backlog in database")

	query, args, err := r.sb.Update("backlogs").
		Set("title", backlog.Title).
		Set("description", backlog.Description).
		Set("completed_at", backlog.CompletedAt).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": backlog.ID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while updating backlog")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("backlog.id", backlog.ID).
			Msg("failed to update backlog in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute update query for backlog")
	}

	log.Debug().
		Int64("backlog.id", backlog.ID).
		Msg("backlog updated successfully in database")
	span.SetStatus(codes.Ok, "backlog updated successfully")
	return nil
}

// DeleteBacklog soft deletes backlog item by its id
// It sets the deleted_at timestamp to the current time
func (r *backlogRepository) DeleteBacklog(ctx context.Context, id int64) error {
	ctx, span := r.tracer.Start(ctx, "backlog_repository.DeleteBacklog",
		trace.WithAttributes(
			attribute.Int64("backlog.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("backlog.id", id).
		Msg("deleting backlog in database")

	query, args, err := r.sb.Update("backlogs").
		Set("deleted_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while deleting backlog")
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("backlog.id", id).
			Msg("failed to delete backlog in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute delete query for backlog")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Debug().
			Err(err).
			Int64("backlog.id", id).
			Msg("failed to get rows affected after deleting backlog")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to get rows affected after deleting backlog")
	}

	if rowsAffected == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no backlog found for passed id")
		log.Debug().
			Err(err).
			Int64("backlog.id", id).
			Msg("backlog not found for deletion")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	log.Debug().
		Int64("backlog.id", id).
		Msg("backlog deleted successfully from database")
	span.SetStatus(codes.Ok, "backlog deleted successfully")
	return nil
}

// GetCompletedBacklogsCount retrieves the count of completed backlogs for a user within a date range
func (r *backlogRepository) GetCompletedBacklogsCount(ctx context.Context, userID int64, startDate, endDate time.Time) (int, error) {
	ctx, span := r.tracer.Start(ctx, "backlog_repository.GetCompletedBacklogsCount",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
			attribute.String("start_date", startDate.Format(time.RFC3339)),
			attribute.String("end_date", endDate.Format(time.RFC3339)),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", userID).
		Str("start_date", startDate.Format(time.RFC3339)).
		Str("end_date", endDate.Format(time.RFC3339)).
		Msg("getting completed backlogs count from database")

	query, args, err := r.sb.Select("COUNT(*)").
		From("backlogs").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.NotEq{"completed_at": nil}).
		Where(squirrel.GtOrEq{"completed_at": startDate}).
		Where(squirrel.LtOrEq{"completed_at": endDate}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while getting completed backlogs count")
	}

	var count int
	err = r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get completed backlogs count from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to get completed backlogs count")
	}

	log.Debug().
		Int64("user.id", userID).
		Int("count", count).
		Msg("completed backlogs count retrieved successfully from database")
	span.SetAttributes(attribute.Int("count", count))
	span.SetStatus(codes.Ok, "completed backlogs count retrieved successfully")
	return count, nil
}
