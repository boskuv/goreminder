package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// CalendarSyncScheduler periodically syncs import bindings and processes the export outbox.
type CalendarSyncScheduler struct {
	syncService *CalendarSyncService
	interval    time.Duration
	logger      zerolog.Logger
}

// NewCalendarSyncScheduler creates a scheduler.
func NewCalendarSyncScheduler(syncService *CalendarSyncService, interval time.Duration, logger zerolog.Logger) *CalendarSyncScheduler {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &CalendarSyncScheduler{
		syncService: syncService,
		interval:    interval,
		logger:      logger,
	}
}

// Start runs until ctx is cancelled.
func (s *CalendarSyncScheduler) Start(ctx context.Context) {
	s.logger.Info().Dur("interval", s.interval).Msg("calendar sync scheduler started")
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("calendar sync scheduler stopped")
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *CalendarSyncScheduler) runOnce(ctx context.Context) {
	bindings, err := s.syncService.bindings.ListActiveForSync(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list calendar bindings for sync")
	} else {
		for _, b := range bindings {
			if err := s.syncService.SyncBinding(ctx, b.ID); err != nil {
				s.logger.Error().Err(err).Int64("calendar_binding.id", b.ID).Msg("binding sync failed")
			}
		}
	}
	if err := s.syncService.ProcessOutbox(ctx); err != nil {
		s.logger.Error().Err(err).Msg("outbox processing failed")
	}
}
