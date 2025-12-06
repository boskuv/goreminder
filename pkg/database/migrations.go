package database

import (
	"fmt"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
)

// RunMigrations runs database migrations from the specified directory
// It uses goose to apply all pending migrations
func RunMigrations(db *sqlx.DB, migrationsDir string, log zerolog.Logger) error {
	// Get the underlying *sql.DB from sqlx.DB
	sqlDB := db.DB

	// Set the dialect for goose
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Get absolute path to migrations directory
	absPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for migrations directory: %w", err)
	}

	log.Info().
		Str("migrations_dir", absPath).
		Msg("running database migrations")

	// Run migrations
	if err := goose.Up(sqlDB, absPath); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get current migration version to log
	currentVersion, err := goose.GetDBVersion(sqlDB)
	if err != nil {
		log.Warn().Err(err).Msg("failed to get current migration version")
	} else {
		log.Info().
			Int64("current_version", currentVersion).
			Msg("database migrations completed successfully")
	}

	return nil
}

// RunMigrationsWithConfig runs migrations using DBConfig to create a temporary connection
// This is useful when you need to run migrations before the main connection is established
func RunMigrationsWithConfig(cfg *DBConfig, migrationsDir string, log zerolog.Logger) error {
	// Create a temporary connection for migrations
	db, err := NewDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database connection for migrations: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close migration database connection")
		}
	}()

	return RunMigrations(db, migrationsDir, log)
}
