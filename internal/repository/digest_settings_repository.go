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

type DigestSettingsRepository interface {
	CreateDigestSettings(ctx context.Context, settings *models.DigestSettings) (int64, error)
	GetDigestSettingsByUserID(ctx context.Context, userID int64, messengerRelatedUserID *int) (*models.DigestSettings, error)
	GetDigestSettingsByID(ctx context.Context, id int64) (*models.DigestSettings, error)
	UpdateDigestSettings(ctx context.Context, settings *models.DigestSettings) error
	DeleteDigestSettings(ctx context.Context, userID int64, messengerRelatedUserID *int) error
	GetAllDigestSettings(ctx context.Context, page, pageSize int, orderBy string, userID *int64) ([]*models.DigestSettings, int, error)
}

type digestSettingsRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewDigestSettingsRepository(db *sqlx.DB, logger zerolog.Logger) DigestSettingsRepository {
	return &digestSettingsRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("digest-settings-repository"),
		logger: logger,
	}
}

// CreateDigestSettings inserts a new digest settings record into the database
func (r *digestSettingsRepository) CreateDigestSettings(ctx context.Context, settings *models.DigestSettings) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "digest_settings_repository.CreateDigestSettings",
		trace.WithAttributes(
			attribute.Int64("user.id", settings.UserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", settings.UserID).
		Msg("creating digest settings in database")

	query, args, err := r.sb.Insert("digest_settings").
		Columns("user_id", "messenger_related_user_id", "enabled", "weekday_time", "weekend_time").
		Values(settings.UserID, settings.MessengerRelatedUserID, settings.Enabled, settings.WeekdayTime, settings.WeekendTime).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while creating new digest settings")
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to insert digest settings")
	}

	span.SetAttributes(attribute.Int64("digest_settings.id", id))
	span.SetStatus(codes.Ok, "digest settings created successfully")
	return id, nil
}

// GetDigestSettingsByUserID retrieves digest settings by user ID and optional messenger related user ID
func (r *digestSettingsRepository) GetDigestSettingsByUserID(ctx context.Context, userID int64, messengerRelatedUserID *int) (*models.DigestSettings, error) {
	ctx, span := r.tracer.Start(ctx, "digest_settings_repository.GetDigestSettingsByUserID",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("getting digest settings by user id from database")

	builder := r.sb.Select("id", "user_id", "messenger_related_user_id", "enabled", "weekday_time", "weekend_time", "created_at", "updated_at").
		From("digest_settings").
		Where(squirrel.Eq{"user_id": userID})

	if messengerRelatedUserID != nil {
		builder = builder.Where(squirrel.Eq{"messenger_related_user_id": *messengerRelatedUserID})
		span.SetAttributes(attribute.Int("messenger_related_user.id", *messengerRelatedUserID))
	} else {
		builder = builder.Where(squirrel.Eq{"messenger_related_user_id": nil})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting digest settings by user id")
	}

	var settings models.DigestSettings
	err = r.db.GetContext(ctx, &settings, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no digest settings found for user")
			log.Debug().
				Err(err).
				Int64("user.id", userID).
				Msg("digest settings not found")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get digest settings by user id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get digest settings by user id")
	}

	log.Debug().
		Int64("user.id", userID).
		Int64("digest_settings.id", settings.ID).
		Msg("digest settings retrieved successfully from database")
	span.SetAttributes(attribute.Int64("digest_settings.id", settings.ID))
	span.SetStatus(codes.Ok, "digest settings retrieved successfully")
	return &settings, nil
}

// GetDigestSettingsByID retrieves digest settings by ID
func (r *digestSettingsRepository) GetDigestSettingsByID(ctx context.Context, id int64) (*models.DigestSettings, error) {
	ctx, span := r.tracer.Start(ctx, "digest_settings_repository.GetDigestSettingsByID",
		trace.WithAttributes(
			attribute.Int64("digest_settings.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("digest_settings.id", id).
		Msg("getting digest settings by id from database")

	query, args, err := r.sb.Select("id", "user_id", "messenger_related_user_id", "enabled", "weekday_time", "weekend_time", "created_at", "updated_at").
		From("digest_settings").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting digest settings by id")
	}

	var settings models.DigestSettings
	err = r.db.GetContext(ctx, &settings, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no digest settings found for id")
			log.Debug().
				Err(err).
				Int64("digest_settings.id", id).
				Msg("digest settings not found")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		log.Debug().
			Err(err).
			Int64("digest_settings.id", id).
			Msg("failed to get digest settings by id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get digest settings by id")
	}

	log.Debug().
		Int64("digest_settings.id", id).
		Msg("digest settings retrieved successfully from database")
	span.SetStatus(codes.Ok, "digest settings retrieved successfully")
	return &settings, nil
}

// UpdateDigestSettings updates digest settings with not nil fields passed in request
// It sets the updated_at to the current time
func (r *digestSettingsRepository) UpdateDigestSettings(ctx context.Context, settings *models.DigestSettings) error {
	ctx, span := r.tracer.Start(ctx, "digest_settings_repository.UpdateDigestSettings",
		trace.WithAttributes(
			attribute.Int64("digest_settings.id", settings.ID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("digest_settings.id", settings.ID).
		Msg("updating digest settings in database")

	query, args, err := r.sb.Update("digest_settings").
		Set("enabled", settings.Enabled).
		Set("weekday_time", settings.WeekdayTime).
		Set("weekend_time", settings.WeekendTime).
		Set("messenger_related_user_id", settings.MessengerRelatedUserID).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": settings.ID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while updating digest settings")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("digest_settings.id", settings.ID).
			Msg("failed to update digest settings in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute update query for digest settings")
	}

	log.Debug().
		Int64("digest_settings.id", settings.ID).
		Msg("digest settings updated successfully in database")
	span.SetStatus(codes.Ok, "digest settings updated successfully")
	return nil
}

// GetAllDigestSettings retrieves all digest settings with pagination, ordering, and filtering
func (r *digestSettingsRepository) GetAllDigestSettings(ctx context.Context, page, pageSize int, orderBy string, userID *int64) ([]*models.DigestSettings, int, error) {
	ctx, span := r.tracer.Start(ctx, "digest_settings_repository.GetAllDigestSettings",
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
		Msg("getting all digest settings from database")

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
		From("digest_settings")

	if userID != nil {
		countBuilder = countBuilder.Where(squirrel.Eq{"user_id": *userID})
		span.SetAttributes(attribute.Int64("user.id", *userID))
	}

	countQuery, countArgs, err := countBuilder.ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to build count query while getting all digest settings")
	}

	var totalCount int
	err = r.db.GetContext(ctx, &totalCount, countQuery, countArgs...)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get total count of digest settings from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get total count of digest settings")
	}

	// Build data query
	dataBuilder := r.sb.Select("id", "user_id", "messenger_related_user_id", "enabled", "weekday_time", "weekend_time", "created_at", "updated_at").
		From("digest_settings")

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
		return nil, 0, errors.Wrap(err, "failed to build query while getting all digest settings")
	}

	var settings []*models.DigestSettings
	err = r.db.SelectContext(ctx, &settings, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all digest settings from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.Wrap(err, "failed to get all digest settings")
	}

	log.Debug().
		Int("settings.count", len(settings)).
		Int("total_count", totalCount).
		Msg("digest settings retrieved successfully from database")
	span.SetAttributes(
		attribute.Int("settings.count", len(settings)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "digest settings retrieved successfully")
	return settings, totalCount, nil
}

// DeleteDigestSettings deletes digest settings by user ID and optional messenger related user ID
func (r *digestSettingsRepository) DeleteDigestSettings(ctx context.Context, userID int64, messengerRelatedUserID *int) error {
	ctx, span := r.tracer.Start(ctx, "digest_settings_repository.DeleteDigestSettings",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("deleting digest settings in database")

	builder := r.sb.Delete("digest_settings").
		Where(squirrel.Eq{"user_id": userID})

	if messengerRelatedUserID != nil {
		builder = builder.Where(squirrel.Eq{"messenger_related_user_id": *messengerRelatedUserID})
		span.SetAttributes(attribute.Int("messenger_related_user.id", *messengerRelatedUserID))
	} else {
		builder = builder.Where(squirrel.Eq{"messenger_related_user_id": nil})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while deleting digest settings")
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to delete digest settings in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute delete query for digest settings")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get rows affected after deleting digest settings")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to get rows affected after deleting digest settings")
	}

	if rowsAffected == 0 {
		err = errors.Wrap(errs.ErrNotFound, "no digest settings found for user")
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("digest settings not found for deletion")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	log.Debug().
		Int64("user.id", userID).
		Msg("digest settings deleted successfully from database")
	span.SetStatus(codes.Ok, "digest settings deleted successfully")
	return nil
}
