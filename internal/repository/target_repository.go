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

type TargetRepository interface {
	CreateTarget(ctx context.Context, target *models.Target) (int64, error)
	GetTargetByID(ctx context.Context, id int64) (*models.Target, error)
	GetAllTargets(ctx context.Context, page, pageSize int, orderBy string, userID *int64) ([]*models.Target, int, error)
	UpdateTarget(ctx context.Context, target *models.Target) error
	DeleteTarget(ctx context.Context, id int64) error
}

type targetRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewTargetRepository(db *sqlx.DB, logger zerolog.Logger) TargetRepository {
	return &targetRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("target-repository"),
		logger: logger,
	}
}

// CreateTarget inserts a new target item into the database
func (r *targetRepository) CreateTarget(ctx context.Context, target *models.Target) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "target_repository.CreateTarget",
		trace.WithAttributes(
			attribute.Int64("user.id", target.UserID),
			attribute.String("target.title", target.Title),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", target.UserID).
		Str("target.title", target.Title).
		Msg("creating target in database")

	query, args, err := r.sb.Insert("targets").
		Columns("title", "description", "user_id", "messenger_related_user_id").
		Values(target.Title, target.Description, target.UserID, target.MessengerRelatedUserID).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while creating new target")
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to insert target")
	}

	span.SetAttributes(attribute.Int64("target.id", id))
	span.SetStatus(codes.Ok, "target created successfully")
	return id, nil
}

// GetTargetByID retrieves a target item by its ID
func (r *targetRepository) GetTargetByID(ctx context.Context, id int64) (*models.Target, error) {
	ctx, span := r.tracer.Start(ctx, "target_repository.GetTargetByID",
		trace.WithAttributes(
			attribute.Int64("target.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("target.id", id).
		Msg("getting target by id from database")

	query, args, err := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "created_at", "updated_at", "completed_at").
		From("targets").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting target by id")
	}

	var target models.Target
	err = r.db.GetContext(ctx, &target, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no target found for passed id")
			log.Debug().
				Err(err).
				Int64("target.id", id).
				Msg("target not found")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		log.Debug().
			Err(err).
			Int64("target.id", id).
			Msg("failed to get target by id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get target by id")
	}

	log.Debug().
		Int64("target.id", id).
		Int64("user.id", target.UserID).
		Msg("target retrieved successfully from database")
	span.SetAttributes(attribute.Int64("user.id", target.UserID))
	span.SetStatus(codes.Ok, "target retrieved successfully")
	return &target, nil
}

// GetAllTargets retrieves all target items with pagination, ordering, and filtering
func (r *targetRepository) GetAllTargets(ctx context.Context, page, pageSize int, orderBy string, userID *int64) ([]*models.Target, int, error) {
	ctx, span := r.tracer.Start(ctx, "target_repository.GetAllTargets",
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
		Msg("getting all targets from database")

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
		From("targets").
		Where(squirrel.Eq{"deleted_at": nil})

	if userID != nil {
		countBuilder = countBuilder.Where(squirrel.Eq{"user_id": *userID})
		span.SetAttributes(attribute.Int64("user.id", *userID))
	}

	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to build count query while getting all targets")
	}

	var totalCount int
	err = r.db.GetContext(ctx, &totalCount, countQuery, countArgs...)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get total count of targets from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get total count of targets")
	}

	// Build data query
	dataBuilder := r.sb.Select("id", "title", "description", "user_id", "messenger_related_user_id", "created_at", "updated_at", "completed_at").
		From("targets").
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
		return nil, 0, errors.Wrap(err, "failed to build query while getting all targets")
	}

	var targets []*models.Target
	err = r.db.SelectContext(ctx, &targets, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all targets from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get all targets")
	}

	log.Debug().
		Int("targets.count", len(targets)).
		Int("total_count", totalCount).
		Msg("targets retrieved successfully from database")
	span.SetAttributes(
		attribute.Int("targets.count", len(targets)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "targets retrieved successfully")
	return targets, totalCount, nil
}

// UpdateTarget updates target item with not nil fields passed in request
// It sets the updated_at to the current time
func (r *targetRepository) UpdateTarget(ctx context.Context, target *models.Target) error {
	ctx, span := r.tracer.Start(ctx, "target_repository.UpdateTarget",
		trace.WithAttributes(
			attribute.Int64("target.id", target.ID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("target.id", target.ID).
		Msg("updating target in database")

	query, args, err := r.sb.Update("targets").
		Set("title", target.Title).
		Set("description", target.Description).
		Set("completed_at", target.CompletedAt).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": target.ID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while updating target")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("target.id", target.ID).
			Msg("failed to update target in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute update query for target")
	}

	log.Debug().
		Int64("target.id", target.ID).
		Msg("target updated successfully in database")
	span.SetStatus(codes.Ok, "target updated successfully")
	return nil
}

// DeleteTarget soft deletes target item by its id
// It sets the deleted_at timestamp to the current time
func (r *targetRepository) DeleteTarget(ctx context.Context, id int64) error {
	ctx, span := r.tracer.Start(ctx, "target_repository.DeleteTarget",
		trace.WithAttributes(
			attribute.Int64("target.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("target.id", id).
		Msg("deleting target in database")

	query, args, err := r.sb.Update("targets").
		Set("deleted_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while deleting target")
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("target.id", id).
			Msg("failed to delete target in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute delete query for target")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Debug().
			Err(err).
			Int64("target.id", id).
			Msg("failed to get rows affected after deleting target")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to get rows affected after deleting target")
	}

	if rowsAffected == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no target found for passed id")
		log.Debug().
			Err(err).
			Int64("target.id", id).
			Msg("target not found for deletion")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	log.Debug().
		Int64("target.id", id).
		Msg("target deleted successfully from database")
	span.SetStatus(codes.Ok, "target deleted successfully")
	return nil
}
