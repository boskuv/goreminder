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

type UserRepository interface {
	CreateUser(user *models.User) (int64, error)
	GetUserByID(id int64) (*models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id int64) error
}

type userRepository struct {
	db *sqlx.DB
	sb squirrel.StatementBuilderType
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{
		db: db,
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// CreateUser inserts a new user into the database
// default values are preset for: id, created_at (database-level)
// nil values are preset for: deleted_at (database-level)
func (r *userRepository) CreateUser(user *models.User) (int64, error) {
	query, args, err := r.sb.Insert("users").
		Columns("name", "email", "password_hash", "timezone").
		Values(user.Name, user.Email, user.PasswordHash, user.Timezone).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to build query while creating new user")
	}

	var id int64
	err = r.db.QueryRow(query, args...).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert user")
	}

	return id, nil
}

// GetUserByID retrieves a user by ID
// Returns user entity and an error if occurred
func (r *userRepository) GetUserByID(id int64) (*models.User, error) {
	query, args, err := r.sb.Select("id", "name", "email", "password_hash", "created_at", "timezone").
		From("users").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query while getting user by id")
	}

	var user models.User
	err = r.db.Get(&user, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Wrap(errs.ErrNotFound, "no user found for passed id")
		}

		return nil, errors.Wrap(err, "failed to get user by id")
	}

	return &user, nil
}

// UpdateUser updates user with not nil fields passed in request
// It sets the updated_at to the current time
func (r *userRepository) UpdateUser(user *models.User) error {
	query, args, err := r.sb.Update("users").
		Set("name", user.Name).
		Set("email", user.Email).
		Set("password_hash", user.PasswordHash).
		Set("timezone", user.Timezone).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": user.ID}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build query while updating user")
	}

	_, err = r.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute update query for user")
	}

	return nil
}

// DeleteUser soft deletes user by its id
// It sets the deleted_at timestamp to the current time
func (r *userRepository) DeleteUser(id int64) error {
	query, args, err := r.sb.Update("users").
		Set("deleted_at", time.Now().UTC()).
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build query while soft deleting user")
	}

	_, err = r.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute soft delete query for user")
	}

	return nil
}
