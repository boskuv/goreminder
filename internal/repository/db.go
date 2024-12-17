package repository

import (
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Import PGX driver for sqlx
	"github.com/jmoiron/sqlx"
)

type DBConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	DbName       string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  int
}

// NewDB initializes and returns a sqlx.DB instance
func NewDB(cfg *DBConfig) (*sqlx.DB, error) {
	// Build the connection string
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DbName, cfg.SSLMode,
	)

	// Initialize the connection
	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
