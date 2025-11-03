package database

import (
	"context"
	"fmt"
	"log"
	"sync"
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
	MaxLifetime  time.Duration
	MaxRetries   int
	RetryDelay   time.Duration // Delay between retries
}

// NewDB initializes and returns a sqlx.DB instance with retry logic
// It will retry connecting to the database if the initial connection fails
func NewDB(cfg *DBConfig) (*sqlx.DB, error) {
	// Set default retry delay if not provided
	retryDelay := cfg.RetryDelay
	if retryDelay == 0 {
		retryDelay = 3 * time.Second
	}

	// build the connection string
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DbName, cfg.SSLMode,
	)

	var db *sqlx.DB
	var err error
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1 // At least try once
	}

	// Retry connection and ping
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("database: retrying connection (attempt %d/%d) to %s:%s/%s",
				attempt+1, maxRetries, cfg.Host, cfg.Port, cfg.DbName)
		} else {
			log.Printf("database: connecting to %s:%s/%s", cfg.Host, cfg.Port, cfg.DbName)
		}

		// initialize the connection
		db, err = sqlx.Open("pgx", dsn)
		if err != nil {
			log.Printf("database: failed to open connection: %v", err)
			if attempt < maxRetries-1 {
				log.Printf("database: waiting %v before retry", retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
		}

		// configure connection pool
		db.SetMaxOpenConns(cfg.MaxOpenConns)
		db.SetMaxIdleConns(cfg.MaxIdleConns)
		db.SetConnMaxLifetime(cfg.MaxLifetime)

		// test the connection
		err = db.Ping()
		if err == nil {
			log.Printf("database: connection established successfully to %s:%s/%s", cfg.Host, cfg.Port, cfg.DbName)
			return db, nil
		}

		log.Printf("database: ping failed: %v", err)
		// If ping failed, close the connection and retry
		db.Close()
		if attempt < maxRetries-1 {
			log.Printf("database: waiting %v before retry", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf("failed to ping database after %d attempts: %w", maxRetries, err)
}

// Reconnect attempts to reconnect to the database and returns a new *sqlx.DB instance
// This should be used when EnsureConnection indicates the connection is lost
// The caller should replace their db variable with the returned one
func Reconnect(cfg *DBConfig) (*sqlx.DB, error) {
	return NewDB(cfg)
}

// EnsureConnection checks if the database connection is alive with retries
// If the connection is lost, it returns an error
// Call Reconnect() if this function returns an error to get a new connection
func EnsureConnection(db *sqlx.DB, cfg *DBConfig) error {
	retryDelay := cfg.RetryDelay
	if retryDelay == 0 {
		retryDelay = 3 * time.Second
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // Default to 3 retries for runtime connection check
	}

	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = db.Ping()
		if err == nil {
			return nil
		}

		// If not the last attempt, wait before retrying
		if attempt < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("database connection is not alive after %d ping attempts: %w", maxRetries, err)
}

// IsConnectionAlive checks if the database connection is currently alive
func IsConnectionAlive(db *sqlx.DB) bool {
	return db.Ping() == nil
}

// ConnectionManager manages the database connection with automatic health checking and reconnection
type ConnectionManager struct {
	config        *DBConfig
	db            *sqlx.DB
	dbMutex       sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	checkInterval time.Duration
	onReconnect   func(*sqlx.DB) // Callback when reconnection happens
}

// NewConnectionManager creates a new connection manager with automatic health checking
// checkInterval: how often to check the connection (default: 30s if 0)
// onReconnect: optional callback function called when reconnection succeeds
func NewConnectionManager(ctx context.Context, cfg *DBConfig, checkInterval time.Duration, onReconnect func(*sqlx.DB)) (*ConnectionManager, error) {
	if checkInterval == 0 {
		checkInterval = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(ctx)

	manager := &ConnectionManager{
		config:        cfg,
		ctx:           ctx,
		cancel:        cancel,
		checkInterval: checkInterval,
		onReconnect:   onReconnect,
	}

	// Initial connection
	log.Printf("database: initializing connection manager for %s:%s/%s", cfg.Host, cfg.Port, cfg.DbName)
	db, err := NewDB(cfg)
	if err != nil {
		cancel()
		log.Printf("database: failed to establish initial connection: %v", err)
		return nil, fmt.Errorf("failed to establish initial database connection: %w", err)
	}

	manager.db = db
	log.Printf("database: connection manager initialized successfully, health checker started (interval: %v)", checkInterval)

	// Start health checker goroutine
	manager.wg.Add(1)
	go manager.healthChecker()

	return manager, nil
}

// GetDB returns the current database connection in a thread-safe manner
func (cm *ConnectionManager) GetDB() *sqlx.DB {
	cm.dbMutex.RLock()
	defer cm.dbMutex.RUnlock()
	return cm.db
}

// healthChecker periodically checks the database connection and reconnects if needed
func (cm *ConnectionManager) healthChecker() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.dbMutex.RLock()
			currentDB := cm.db
			cm.dbMutex.RUnlock()

			// Check if connection is alive
			// Note: sqlx.DB handles reconnection automatically on queries, but we check
			// proactively to detect connection issues early
			if err := EnsureConnection(currentDB, cm.config); err != nil {
				// Connection appears to be lost
				log.Printf("database: connection check failed - connection lost: %v", err)
				log.Printf("database: attempting to reconnect to %s:%s/%s",
					cm.config.Host, cm.config.Port, cm.config.DbName)

				// sqlx.DB will automatically reconnect on the next query attempt,
				// but if the pool is completely broken, we create a new instance
				// This is only needed in extreme cases where the pool cannot recover

				// Try to reconnect by creating a new DB instance
				newDB, reconnectErr := Reconnect(cm.config)
				if reconnectErr != nil {
					// Reconnection failed - sqlx.DB will try on next query
					log.Printf("database: reconnection failed: %v (will retry on next health check)", reconnectErr)
					continue
				}

				log.Printf("database: reconnection successful to %s:%s/%s",
					cm.config.Host, cm.config.Port, cm.config.DbName)

				// Update connection atomically
				// Note: We replace the DB object, but DON'T close the old one immediately
				// because repositories may still have references to it and have ongoing queries.
				// sqlx.DB.Close() gracefully closes connections after active queries finish,
				// but closing it here would break repositories with active references.
				//
				// The old DB will eventually be garbage collected, and its connection pool
				// will naturally close unused connections. This is acceptable because:
				// 1. We only replace DB when connection is completely broken
				// 2. New queries will use the new DB from GetDB()
				// 3. Old DB references will eventually become inactive
				cm.dbMutex.Lock()
				cm.db = newDB
				cm.dbMutex.Unlock()

				// We intentionally DON'T close oldDB here to avoid breaking active repositories

				// Notify callback
				if cm.onReconnect != nil {
					cm.onReconnect(newDB)
				}
			}
		}
	}
}

// Close gracefully shuts down the connection manager
func (cm *ConnectionManager) Close() error {
	log.Printf("database: shutting down connection manager for %s:%s/%s",
		cm.config.Host, cm.config.Port, cm.config.DbName)

	// Signal shutdown
	cm.cancel()

	// Wait for health checker to finish
	log.Printf("database: waiting for health checker to stop")
	cm.wg.Wait()
	log.Printf("database: health checker stopped")

	// Close database connection
	cm.dbMutex.Lock()
	defer cm.dbMutex.Unlock()

	var err error
	if cm.db != nil {
		log.Printf("database: closing database connection")
		err = cm.db.Close()
		if err != nil {
			log.Printf("database: error closing database connection: %v", err)
		} else {
			log.Printf("database: database connection closed successfully")
		}
	}

	log.Printf("database: connection manager shutdown complete")
	return err
}
