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

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) (int64, error)
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id int64) error
}

type userRepository struct {
	db     *sqlx.DB
	sb     squirrel.StatementBuilderType
	tracer trace.Tracer
	logger zerolog.Logger
}

func NewUserRepository(db *sqlx.DB, logger zerolog.Logger) UserRepository {
	return &userRepository{
		db:     db,
		sb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		tracer: otel.Tracer("user-repository"),
		logger: logger,
	}
}

// CreateUser inserts a new user into the database
// default values are preset for: id, created_at (database-level)
// nil values are preset for: deleted_at (database-level)
func (r *userRepository) CreateUser(ctx context.Context, user *models.User) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "user_repository.CreateUser",
		trace.WithAttributes(
			attribute.String("user.name", user.Name),
			attribute.String("user.email", user.Email),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Str("user.name", user.Name).
		Str("user.email", user.Email).
		Msg("creating user in database")

	query, args, err := r.sb.Insert("users").
		Columns("name", "email", "password_hash", "timezone", "language_code", "role").
		Values(user.Name, user.Email, user.PasswordHash, user.Timezone, user.LanguageCode, user.Role).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to build query while creating new user")
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.Wrap(err, "failed to insert user")
	}

	span.SetAttributes(attribute.Int64("user.id", id))
	span.SetStatus(codes.Ok, "user created successfully")
	return id, nil
}

// GetUserByID retrieves a user by ID
// Returns user entity and an error if occurred
func (r *userRepository) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	ctx, span := r.tracer.Start(ctx, "user_repository.GetUserByID",
		trace.WithAttributes(
			attribute.Int64("user.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", id).
		Msg("getting user by id from database")

	query, args, err := r.sb.Select("id", "name", "email", "password_hash", "created_at", "timezone", "language_code", "role").
		From("users").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to build query while getting user by id")
	}

	var user models.User
	err = r.db.GetContext(ctx, &user, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Wrap(errs.ErrNotFound, "no user found for passed id")
			log.Debug().
				Err(err).
				Int64("user.id", id).
				Msg("user not found")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		log.Debug().
			Err(err).
			Int64("user.id", id).
			Msg("failed to get user by id from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to get user by id")
	}

	log.Debug().
		Int64("user.id", id).
		Str("user.name", user.Name).
		Msg("user retrieved successfully from database")
	span.SetStatus(codes.Ok, "user retrieved successfully")
	return &user, nil
}

// UpdateUser updates user with not nil fields passed in request
// It sets the updated_at to the current time
func (r *userRepository) UpdateUser(ctx context.Context, user *models.User) error {
	ctx, span := r.tracer.Start(ctx, "user_repository.UpdateUser",
		trace.WithAttributes(
			attribute.Int64("user.id", user.ID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", user.ID).
		Msg("updating user in database")

	query, args, err := r.sb.Update("users").
		Set("name", user.Name).
		Set("email", user.Email).
		Set("password_hash", user.PasswordHash).
		Set("timezone", user.Timezone).
		Set("language_code", user.LanguageCode).
		Set("role", user.Role).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": user.ID}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while updating user")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", user.ID).
			Msg("failed to update user in database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute update query for user")
	}

	log.Debug().
		Int64("user.id", user.ID).
		Msg("user updated successfully in database")
	span.SetStatus(codes.Ok, "user updated successfully")
	return nil
}

// DeleteUser soft deletes user by its id
// It sets the deleted_at timestamp to the current time
func (r *userRepository) DeleteUser(ctx context.Context, id int64) error {
	ctx, span := r.tracer.Start(ctx, "user_repository.DeleteUser",
		trace.WithAttributes(
			attribute.Int64("user.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, r.logger)
	log.Debug().
		Int64("user.id", id).
		Msg("deleting user from database")

	query, args, err := r.sb.Update("users").
		Set("deleted_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to build query while soft deleting user")
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", id).
			Msg("failed to delete user from database")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to execute soft delete query for user")
	}

	log.Debug().
		Int64("user.id", id).
		Msg("user deleted successfully from database")
	span.SetStatus(codes.Ok, "user deleted successfully")
	return nil
}
