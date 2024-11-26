// internal/repository/user_repository.go
package repository

import (
	"database/sql"
	"fmt"
)

// User represents a user in the system
type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    string
}

// UserRepository handles CRUD operations for users
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts a new user into the database
func (r *UserRepository) CreateUser(user *User) (int64, error) {
	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3) RETURNING id
	`

	var id int64
	err := r.db.QueryRow(query, user.Name, user.Email, user.PasswordHash).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return id, nil
}

// GetUserByID retrieves a user by their ID
func (r *UserRepository) GetUserByID(id int64) (*User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM users
		WHERE id = $1
	`

	user := &User{}
	err := r.db.QueryRow(query, id).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user by id: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by their email address
func (r *UserRepository) GetUserByEmail(email string) (*User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM users
		WHERE email = $1
	`

	user := &User{}
	err := r.db.QueryRow(query, email).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user by email: %w", err)
	}

	return user, nil
}

// UpdateUser updates an existing user's details
func (r *UserRepository) UpdateUser(user *User) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3
		WHERE id = $4
	`

	_, err := r.db.Exec(query, user.Name, user.Email, user.PasswordHash, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser deletes a user by their ID
func (r *UserRepository) DeleteUser(id int64) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
