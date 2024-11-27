package repository

import (
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/boskuv/goreminder/internal/models"
)

type UserRepository struct {
	db *sqlx.DB
	sb squirrel.StatementBuilderType
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{
		db: db,
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// CreateUser inserts a new user into the database
func (r *UserRepository) CreateUser(user *models.User) (int64, error) {
	query, args, err := r.sb.Insert("users").
		Columns("name", "email", "password_hash").
		Values(user.Name, user.Email, user.PasswordHash).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build query: %w", err)
	}

	var id int64
	err = r.db.QueryRow(query, args...).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert user: %w", err)
	}

	return id, nil
}

// GetUserByID retrieves a user by their ID
func (r *UserRepository) GetUserByID(id int64) (*models.User, error) {
	query, args, err := r.sb.Select("id", "name", "email", "password_hash", "created_at").
		From("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var user models.User
	err = r.db.Get(&user, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	return &user, nil
}
