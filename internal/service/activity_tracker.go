package service

import (
	"context"
	"time"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/rs/zerolog"
)

// ActivityTracker records and lists user-driven last-activity timestamps.
type ActivityTracker interface {
	Touch(ctx context.Context, userID int64, at time.Time) error
	ListRecent(ctx context.Context, limit int) ([]models.UserActivity, error)
}

// NoopActivityTracker discards Touch and returns an empty list from ListRecent.
type NoopActivityTracker struct{}

func (NoopActivityTracker) Touch(context.Context, int64, time.Time) error { return nil }

func (NoopActivityTracker) ListRecent(context.Context, int) ([]models.UserActivity, error) {
	return []models.UserActivity{}, nil
}

// PostgresActivityTracker persists activity on users.last_activity_at.
type PostgresActivityTracker struct {
	userRepo repository.UserRepository
	logger   zerolog.Logger
}

// NewPostgresActivityTracker creates an ActivityTracker backed by UserRepository.
func NewPostgresActivityTracker(userRepo repository.UserRepository, logger zerolog.Logger) *PostgresActivityTracker {
	return &PostgresActivityTracker{userRepo: userRepo, logger: logger}
}

func (t *PostgresActivityTracker) Touch(ctx context.Context, userID int64, at time.Time) error {
	return t.userRepo.TouchLastActivity(ctx, userID, at)
}

func (t *PostgresActivityTracker) ListRecent(ctx context.Context, limit int) ([]models.UserActivity, error) {
	return t.userRepo.ListRecentActivity(ctx, limit)
}

// touchUserActivity best-effort records last activity; failures are logged and ignored.
func touchUserActivity(ctx context.Context, tracker ActivityTracker, log zerolog.Logger, userID int64) {
	if tracker == nil || userID <= 0 {
		return
	}
	if err := tracker.Touch(ctx, userID, time.Now().UTC()); err != nil {
		traced := logger.WithTraceContext(ctx, log)
		traced.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to record user activity")
	}
}
