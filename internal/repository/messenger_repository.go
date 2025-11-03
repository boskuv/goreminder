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

type MessengerRepository interface {
	CreateMessenger(ctx context.Context, messenger *models.Messenger) (int64, error)
	GetMessengerByID(ctx context.Context, id int64) (*models.Messenger, error)
	GetMessengerIDByName(ctx context.Context, messengerName string) (int64, error)
	CreateMessengerRelatedUser(ctx context.Context, messengerRelatedUser *models.MessengerRelatedUser) (int64, error)
	GetMessengerRelatedUser(ctx context.Context, chatID string, messengerUserID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error)
	GetUserID(ctx context.Context, messengerUserID string) (int64, error)
	GetMessengerRelatedUserByID(ctx context.Context, messengerUserID int) (*models.MessengerRelatedUser, error)
	DeleteMessengerRelatedUserByUserID(ctx context.Context, userID int64) error
}

type messengerRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewMessengerRepository(db *sqlx.DB, logger zerolog.Logger) MessengerRepository {
	return &messengerRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("messenger-repository"),
		logger: logger,
	}
}

// CreateMessenger inserts a new messenger into the database
// default values are preset for: id, created_at (database-level)
func (r *messengerRepository) CreateMessenger(ctx context.Context, messenger *models.Messenger) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "messenger_repository.CreateMessenger",
		trace.WithAttributes(
			attribute.String("messenger.name", messenger.Name),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Str("messenger.name", messenger.Name).
		Msg("creating messenger in database")

	query, args, err := r.sb.Insert("messengers").
		Columns("name").
		Values(messenger.Name).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while creating new messenger")
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		log.Debug().
			Err(err).
			Str("messenger.name", messenger.Name).
			Msg("failed to create messenger in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to insert messenger")
	}

	log.Debug().
		Int64("messenger.id", id).
		Str("messenger.name", messenger.Name).
		Msg("messenger created successfully in database")
	span.SetAttributes(attribute.Int64("messenger.id", id))
	span.SetStatus(codes.Ok, "messenger created successfully")
	return id, nil
}

// GetMessengerByID retrieves a messenger by its ID
// Returns messenger entity and an error if occurred
func (r *messengerRepository) GetMessengerByID(ctx context.Context, id int64) (*models.Messenger, error) {
	ctx, span := r.tracer.Start(ctx, "messenger_repository.GetMessengerByID",
		trace.WithAttributes(
			attribute.Int64("messenger.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("messenger.id", id).
		Msg("getting messenger by id from database")

	query, args, err := r.sb.Select("name", "created_at").
		From("messengers").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting messenger by id")
	}

	var messenger models.Messenger
	err = r.db.GetContext(ctx, &messenger, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no messenger found for passed id")
			log.Debug().
				Err(err).
				Int64("messenger.id", id).
				Msg("messenger not found")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		log.Debug().
			Err(err).
			Int64("messenger.id", id).
			Msg("failed to get messenger by id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get messenger by id")
	}

	log.Debug().
		Int64("messenger.id", id).
		Str("messenger.name", messenger.Name).
		Msg("messenger retrieved successfully from database")
	span.SetStatus(codes.Ok, "messenger retrieved successfully")
	return &messenger, nil
}

// GetMessengerIDByName retrieves a messenger ID by its name
// Returns messenger ID and an error if occurred
func (r *messengerRepository) GetMessengerIDByName(ctx context.Context, messengerName string) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "messenger_repository.GetMessengerIDByName",
		trace.WithAttributes(
			attribute.String("messenger.name", messengerName),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Str("messenger.name", messengerName).
		Msg("getting messenger id by name from database")

	query, args, err := r.sb.Select("id").
		From("messengers").
		Where(squirrel.Eq{"name": messengerName}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while getting messenger id by name")
	}

	var messengerID int64
	err = r.db.GetContext(ctx, &messengerID, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no messenger found for passed name")
			log.Debug().
				Err(err).
				Str("messenger.name", messengerName).
				Msg("messenger not found by name")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return 0, err
		}

		log.Debug().
			Err(err).
			Str("messenger.name", messengerName).
			Msg("failed to get messenger id by name from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to get messenger id by name")
	}

	log.Debug().
		Int64("messenger.id", messengerID).
		Str("messenger.name", messengerName).
		Msg("messenger id retrieved successfully from database")
	span.SetAttributes(attribute.Int64("messenger.id", messengerID))
	span.SetStatus(codes.Ok, "messenger id retrieved successfully")
	return messengerID, nil
}

// CreateMessengerRelatedUser inserts a new messenger-related user into the database
// default values are preset for: id, created_at (database-level)
func (r *messengerRepository) CreateMessengerRelatedUser(ctx context.Context, messengerRelatedUser *models.MessengerRelatedUser) (int64, error) {
	attrs := []attribute.KeyValue{
		attribute.String("messenger_user.id", messengerRelatedUser.MessengerUserID),
	}
	if messengerRelatedUser.UserID != nil {
		attrs = append(attrs, attribute.Int64("user.id", *messengerRelatedUser.UserID))
	}
	if messengerRelatedUser.MessengerID != nil {
		attrs = append(attrs, attribute.Int64("messenger.id", *messengerRelatedUser.MessengerID))
	}

	ctx, span := r.tracer.Start(ctx, "messenger_repository.CreateMessengerRelatedUser",
		trace.WithAttributes(attrs...))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Str("messenger_user.id", messengerRelatedUser.MessengerUserID).
		Msg("creating messenger-related user in database")
	if messengerRelatedUser.UserID != nil {
		log = log.With().Int64("user.id", *messengerRelatedUser.UserID).Logger()
	}
	if messengerRelatedUser.MessengerID != nil {
		log = log.With().Int64("messenger.id", *messengerRelatedUser.MessengerID).Logger()
	}

	query, args, err := r.sb.Insert("user_messengers").
		Columns("chat_id", "messenger_id", "messenger_user_id", "user_id").
		Values(messengerRelatedUser.ChatID, messengerRelatedUser.MessengerID, messengerRelatedUser.MessengerUserID, messengerRelatedUser.UserID).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while creating new messenger-related user")
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		log.Debug().
			Err(err).
			Str("messenger_user.id", messengerRelatedUser.MessengerUserID).
			Msg("failed to create messenger-related user in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to insert messenger-related user")
	}

	log.Debug().
		Int64("messenger_related_user.id", id).
		Str("messenger_user.id", messengerRelatedUser.MessengerUserID).
		Msg("messenger-related user created successfully in database")
	span.SetAttributes(attribute.Int64("messenger_related_user.id", id))
	span.SetStatus(codes.Ok, "messenger-related user created successfully")
	return id, nil
}

// GetMessengerRelatedUser retrieves a messenger-related user by chatID, messengerUserID, userID and messengerID
// Returns messenger-related user entity and an error if occurred
func (r *messengerRepository) GetMessengerRelatedUser(ctx context.Context, chatID string, messengerUserID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error) {
	ctx, span := r.tracer.Start(ctx, "messenger_repository.GetMessengerRelatedUser",
		trace.WithAttributes(
			attribute.String("chat.id", chatID),
			attribute.String("messenger_user.id", messengerUserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Str("chat.id", chatID).
		Str("messenger_user.id", messengerUserID).
		Msg("getting messenger-related user from database")
	if userID != nil {
		span.SetAttributes(attribute.Int64("user.id", *userID))
		log = log.With().Int64("user.id", *userID).Logger()
	}
	if messengerID != nil {
		span.SetAttributes(attribute.Int64("messenger.id", *messengerID))
		log = log.With().Int64("messenger.id", *messengerID).Logger()
	}

	query, args, err := r.sb.Select("id", "user_id", "messenger_id", "messenger_user_id", "chat_id", "created_at", "updated_at").
		From("user_messengers").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"chat_id": chatID}).
		Where(squirrel.Eq{"messenger_user_id": messengerUserID}).
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.Eq{"messenger_id": messengerID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting messenger-related user")
	}

	var messengerRelatedUser models.MessengerRelatedUser
	err = r.db.GetContext(ctx, &messengerRelatedUser, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no messenger-related user found for passed chatID, messengerUserID, userID and messengerID")
			log.Debug().
				Err(err).
				Str("chat.id", chatID).
				Str("messenger_user.id", messengerUserID).
				Msg("messenger-related user not found")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		log.Debug().
			Err(err).
			Str("chat.id", chatID).
			Str("messenger_user.id", messengerUserID).
			Msg("failed to get messenger-related user from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get messenger-related user")
	}

	log.Debug().
		Int64("messenger_related_user.id", messengerRelatedUser.ID).
		Str("chat.id", chatID).
		Str("messenger_user.id", messengerUserID).
		Msg("messenger-related user retrieved successfully from database")
	span.SetAttributes(attribute.Int64("messenger_related_user.id", messengerRelatedUser.ID))
	span.SetStatus(codes.Ok, "messenger-related user retrieved successfully")
	return &messengerRelatedUser, nil
}

// GetUserID retrieves a userID user by messengerUserID
// Returns userID and an error if occurred
func (r *messengerRepository) GetUserID(ctx context.Context, messengerUserID string) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "messenger_repository.GetUserID",
		trace.WithAttributes(
			attribute.String("messenger_user.id", messengerUserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Str("messenger_user.id", messengerUserID).
		Msg("getting user id by messenger user id from database")

	query, args, err := r.sb.Select("user_id").
		From("user_messengers").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"messenger_user_id": messengerUserID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while getting user id by messenger user id")
	}

	var userID int64
	err = r.db.GetContext(ctx, &userID, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no user found for passed messenger user id")
			log.Debug().
				Err(err).
				Str("messenger_user.id", messengerUserID).
				Msg("user not found by messenger user id")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return 0, err
		}

		log.Debug().
			Err(err).
			Str("messenger_user.id", messengerUserID).
			Msg("failed to get user id by messenger user id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to get user id by messenger user id")
	}

	log.Debug().
		Int64("user.id", userID).
		Str("messenger_user.id", messengerUserID).
		Msg("user id retrieved successfully from database")
	span.SetAttributes(attribute.Int64("user.id", userID))
	span.SetStatus(codes.Ok, "user id retrieved successfully")
	return userID, nil
}

// GetMessengerRelatedUserByID retrieves a messenger-related user by its ID
// Returns messenger-related user entity and an error if occurred
func (r *messengerRepository) GetMessengerRelatedUserByID(ctx context.Context, messengerUserID int) (*models.MessengerRelatedUser, error) {
	ctx, span := r.tracer.Start(ctx, "messenger_repository.GetMessengerRelatedUserByID",
		trace.WithAttributes(
			attribute.Int("messenger_related_user.id", messengerUserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int("messenger_related_user.id", messengerUserID).
		Msg("getting messenger-related user by id from database")

	query, args, err := r.sb.Select("id", "user_id", "messenger_id", "messenger_user_id", "chat_id", "created_at", "updated_at").
		From("user_messengers").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": messengerUserID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting messenger-related user by id")
	}

	var messengerRelatedUser models.MessengerRelatedUser
	err = r.db.GetContext(ctx, &messengerRelatedUser, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no messenger-related user found for passed id")
			log.Debug().
				Err(err).
				Int("messenger_related_user.id", messengerUserID).
				Msg("messenger-related user not found")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		log.Debug().
			Err(err).
			Int("messenger_related_user.id", messengerUserID).
			Msg("failed to get messenger-related user by id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get messenger-related user by id")
	}

	if messengerRelatedUser.UserID != nil {
		span.SetAttributes(attribute.Int64("user.id", *messengerRelatedUser.UserID))
		log = log.With().Int64("user.id", *messengerRelatedUser.UserID).Logger()
	}
	log.Debug().
		Int("messenger_related_user.id", messengerUserID).
		Msg("messenger-related user retrieved successfully from database")
	span.SetStatus(codes.Ok, "messenger-related user retrieved successfully")
	return &messengerRelatedUser, nil
}

// DeleteMessengerRelatedUserByUserID soft deletes messenger-related user by user id
// It sets the deleted_at timestamp to the current time
func (r *messengerRepository) DeleteMessengerRelatedUserByUserID(ctx context.Context, userID int64) error {
	ctx, span := r.tracer.Start(ctx, "messenger_repository.DeleteMessengerRelatedUserByUserID",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("deleting messenger-related user by user id from database")

	query, args, err := r.sb.Update("user_messengers").
		Set("deleted_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while soft deleting messenger-related user")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to delete messenger-related user by user id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute soft delete query for messenger-related user")
	}

	log.Debug().
		Int64("user.id", userID).
		Msg("messenger-related user deleted successfully from database")
	span.SetStatus(codes.Ok, "messenger-related user deleted successfully")
	return nil
}
