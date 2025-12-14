package service

import (
	"context"
	"time"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/boskuv/goreminder/pkg/queue"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// DigestService defines methods for digest-related business logic
type DigestService struct {
	digestSettingsRepo repository.DigestSettingsRepository
	backlogRepo        repository.BacklogRepository
	taskRepo           repository.TaskRepository
	userRepo           repository.UserRepository
	messengerRepo      repository.MessengerRepository
	producer           *queue.Producer
	tracer             trace.Tracer
	logger             zerolog.Logger
}

// NewDigestService creates a new DigestService
func NewDigestService(
	digestSettingsRepo repository.DigestSettingsRepository,
	backlogRepo repository.BacklogRepository,
	taskRepo repository.TaskRepository,
	userRepo repository.UserRepository,
	messengerRepo repository.MessengerRepository,
	producer *queue.Producer,
	logger zerolog.Logger,
) *DigestService {
	return &DigestService{
		digestSettingsRepo: digestSettingsRepo,
		backlogRepo:        backlogRepo,
		taskRepo:           taskRepo,
		userRepo:           userRepo,
		messengerRepo:      messengerRepo,
		producer:           producer,
		tracer:             otel.Tracer("digest-service"),
		logger:             logger,
	}
}

// CreateDigestSettings implements BL of creating new digest settings
func (s *DigestService) CreateDigestSettings(ctx context.Context, settings *models.DigestSettings) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "digest_service.CreateDigestSettings",
		trace.WithAttributes(
			attribute.Int64("user.id", settings.UserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", settings.UserID).
		Msg("starting digest settings creation")

	// Check if user exists
	_, err := s.userRepo.GetUserByID(ctx, settings.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		log.Debug().
			Err(err).
			Int64("user.id", settings.UserID).
			Msg("user not found or error retrieving user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	if settings.MessengerRelatedUserID != nil {
		span.SetAttributes(attribute.Int("messenger_related_user.id", *settings.MessengerRelatedUserID))
		// Check if messenger related user exists
		_, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *settings.MessengerRelatedUserID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return 0, errors.WithStack(err)
		}
	}

	// Validate time format (HH:MM)
	if err := validateTimeFormat(settings.WeekdayTime); err != nil {
		err = errors.Wrap(errs.ErrValidation, err.Error())
		log.Debug().
			Err(err).
			Str("weekday_time", settings.WeekdayTime).
			Msg("invalid weekday time format")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	if err := validateTimeFormat(settings.WeekendTime); err != nil {
		err = errors.Wrap(errs.ErrValidation, err.Error())
		log.Debug().
			Err(err).
			Str("weekend_time", settings.WeekendTime).
			Msg("invalid weekend time format")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	// Check if settings already exist for this user and messenger
	_, err = s.digestSettingsRepo.GetDigestSettingsByUserID(ctx, settings.UserID, settings.MessengerRelatedUserID)
	if err == nil {
		// Settings already exist
		err = errors.Wrap(errs.ErrUnprocessableEntity, "digest settings already exist for this user and messenger")
		log.Debug().
			Err(err).
			Int64("user.id", settings.UserID).
			Msg("digest settings already exist")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}
	if !errors.Is(err, errs.ErrNotFound) {
		// Some other error occurred
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	log.Debug().
		Int64("user.id", settings.UserID).
		Msg("creating digest settings in repository")
	settingsID, err := s.digestSettingsRepo.CreateDigestSettings(ctx, settings)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", settings.UserID).
			Msg("failed to create digest settings in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	// Publish create to RabbitMQ
	digestQueueMessage := map[string]interface{}{
		"task": "worker.create_digest_settings",
		"args": []interface{}{
			settings.UserID,
			settings.MessengerRelatedUserID,
			settings.Enabled,
			settings.WeekdayTime,
			settings.WeekendTime,
		},
	}

	err = s.producer.Publish(ctx, digestQueueMessage)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", settings.UserID).
			Msg("failed to queue create_digest_settings message")
		// Don't fail the operation, just log the error
		// The database creation was successful, queue update failure is non-critical
	} else {
		log.Debug().
			Int64("user.id", settings.UserID).
			Msg("create_digest_settings message queued successfully")
	}

	span.SetAttributes(attribute.Int64("digest_settings.id", settingsID))
	log.Debug().
		Int64("digest_settings.id", settingsID).
		Int64("user.id", settings.UserID).
		Msg("digest settings created successfully")

	span.SetStatus(codes.Ok, "digest settings created successfully")
	return settingsID, nil
}

// GetDigestSettings implements BL of retrieving digest settings by user ID
func (s *DigestService) GetDigestSettings(ctx context.Context, userID int64, messengerRelatedUserID *int) (*models.DigestSettings, error) {
	ctx, span := s.tracer.Start(ctx, "digest_service.GetDigestSettings",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("getting digest settings")

	settings, err := s.digestSettingsRepo.GetDigestSettingsByUserID(ctx, userID, messengerRelatedUserID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get digest settings")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("user.id", userID).
		Int64("digest_settings.id", settings.ID).
		Msg("digest settings retrieved successfully")
	span.SetStatus(codes.Ok, "digest settings retrieved successfully")
	return settings, nil
}

// UpdateDigestSettings implements BL of updating digest settings
func (s *DigestService) UpdateDigestSettings(ctx context.Context, userID int64, messengerRelatedUserID *int, updateRequest *models.DigestSettingsUpdateRequest) (*models.DigestSettings, error) {
	ctx, span := s.tracer.Start(ctx, "digest_service.UpdateDigestSettings",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("updating digest settings")

	// Get existing settings
	oldSettings, err := s.digestSettingsRepo.GetDigestSettingsByUserID(ctx, userID, messengerRelatedUserID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get digest settings for update")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Update fields if provided
	if updateRequest.Enabled != nil {
		oldSettings.Enabled = *updateRequest.Enabled
	}

	if updateRequest.WeekdayTime != nil {
		if err := validateTimeFormat(*updateRequest.WeekdayTime); err != nil {
			err = errors.Wrap(errs.ErrValidation, err.Error())
			log.Debug().
				Err(err).
				Str("weekday_time", *updateRequest.WeekdayTime).
				Msg("invalid weekday time format in update request")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
		oldSettings.WeekdayTime = *updateRequest.WeekdayTime
	}

	if updateRequest.WeekendTime != nil {
		if err := validateTimeFormat(*updateRequest.WeekendTime); err != nil {
			err = errors.Wrap(errs.ErrValidation, err.Error())
			log.Debug().
				Err(err).
				Str("weekend_time", *updateRequest.WeekendTime).
				Msg("invalid weekend time format in update request")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
		oldSettings.WeekendTime = *updateRequest.WeekendTime
	}

	if updateRequest.MessengerRelatedUserID != nil {
		// Check if messenger related user exists
		_, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *updateRequest.MessengerRelatedUserID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
		oldSettings.MessengerRelatedUserID = updateRequest.MessengerRelatedUserID
	}

	// Update in repository
	err = s.digestSettingsRepo.UpdateDigestSettings(ctx, oldSettings)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to update digest settings in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Publish update to RabbitMQ
	digestQueueMessage := map[string]interface{}{
		"task": "worker.update_digest_settings",
		"args": []interface{}{
			oldSettings.UserID,
			oldSettings.MessengerRelatedUserID,
			oldSettings.Enabled,
			oldSettings.WeekdayTime,
			oldSettings.WeekendTime,
		},
	}

	err = s.producer.Publish(ctx, digestQueueMessage)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to queue update_digest_settings message")
		// Don't fail the operation, just log the error
		// The database update was successful, queue update failure is non-critical
	} else {
		log.Debug().
			Int64("user.id", userID).
			Msg("update_digest_settings message queued successfully")
	}

	log.Debug().
		Int64("user.id", userID).
		Msg("digest settings updated successfully")
	span.SetStatus(codes.Ok, "digest settings updated successfully")
	return oldSettings, nil
}

// GetDigest generates a digest for a user with statistics
func (s *DigestService) GetDigest(ctx context.Context, userID int64, messengerRelatedUserID *int, startDateFrom, startDateTo *time.Time) (*DigestResponse, error) {
	ctx, span := s.tracer.Start(ctx, "digest_service.GetDigest",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("generating digest")

	// Check if user exists
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("user not found or error retrieving user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Get user timezone or use UTC (for default dates only)
	location := time.UTC
	if user.Timezone != nil && *user.Timezone != "" {
		loc, err := time.LoadLocation(*user.Timezone)
		if err != nil {
			log.Warn().
				Err(err).
				Str("timezone", *user.Timezone).
				Msg("failed to load timezone, using UTC")
		} else {
			location = loc
		}
	}

	// Use dates as-is (client sends in UTC)
	var startDateFromInTZ, startDateToInTZ *time.Time
	if startDateFrom != nil {
		startDateFromInTZ = startDateFrom
		span.SetAttributes(attribute.String("start_date_from", startDateFrom.Format(time.RFC3339)))
	}
	if startDateTo != nil {
		startDateToInTZ = startDateTo
		span.SetAttributes(attribute.String("start_date_to", startDateTo.Format(time.RFC3339)))
	}

	// Get completed backlogs count for the period
	var completedCount int
	if startDateFromInTZ != nil && startDateToInTZ != nil {
		completedCount, err = s.backlogRepo.GetCompletedBacklogsCount(ctx, userID, *startDateFromInTZ, *startDateToInTZ)
		if err != nil {
			log.Debug().
				Err(err).
				Int64("user.id", userID).
				Msg("failed to get completed backlogs count")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
	}

	// Get tasks for the period
	tasks, _, err := s.taskRepo.GetTasksByUserIDWithPagination(ctx, userID, 1, 1000, "start_date ASC", startDateFromInTZ, startDateToInTZ, nil, nil, nil)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get tasks")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Set default dates if not provided
	var startDateFromFinal, startDateToFinal time.Time
	if startDateFromInTZ != nil {
		startDateFromFinal = *startDateFromInTZ
	} else {
		startDateFromFinal = time.Now().In(location).Truncate(24 * time.Hour)
	}
	if startDateToInTZ != nil {
		startDateToFinal = *startDateToInTZ
	} else {
		startDateToFinal = time.Now().In(location)
	}

	// Get chat_id if messengerRelatedUserID is provided
	var chatID *string
	if messengerRelatedUserID != nil {
		mru, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *messengerRelatedUserID)
		if err != nil {
			log.Debug().
				Err(err).
				Int("messenger_related_user.id", *messengerRelatedUserID).
				Msg("failed to get messenger related user for chat_id")
			// Don't fail the operation, just log the error
			// chat_id will remain nil
		} else {
			chatID = &mru.ChatID
		}
	}

	digest := &DigestResponse{
		UserID:                 userID,
		MessengerRelatedUserID: messengerRelatedUserID,
		ChatID:                 chatID,
		StartDateFrom:          startDateFromFinal,
		StartDateTo:            startDateToFinal,
		CompletedBacklogsCount: completedCount,
		Tasks:                  tasks,
		Timezone:               location.String(),
	}

	log.Debug().
		Int64("user.id", userID).
		Int("completed_backlogs_count", completedCount).
		Int("tasks.count", len(tasks)).
		Msg("digest generated successfully")
	span.SetAttributes(
		attribute.Int("completed_backlogs_count", completedCount),
		attribute.Int("tasks.count", len(tasks)),
	)
	span.SetStatus(codes.Ok, "digest generated successfully")
	return digest, nil
}

// GetAllDigestSettings implements BL of retrieving all digest settings with pagination
func (s *DigestService) GetAllDigestSettings(ctx context.Context, page, pageSize int, orderBy string, userID *int64) ([]*models.DigestSettings, int, error) {
	ctx, span := s.tracer.Start(ctx, "digest_service.GetAllDigestSettings",
		trace.WithAttributes(
			attribute.Int("page", page),
			attribute.Int("page_size", pageSize),
			attribute.String("order_by", orderBy),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int("page", page).
		Int("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting all digest settings")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	settings, totalCount, err := s.digestSettingsRepo.GetAllDigestSettings(ctx, page, pageSize, orderBy, userID)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all digest settings")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.WithStack(err)
	}

	log.Debug().
		Int("settings.count", len(settings)).
		Int("total_count", totalCount).
		Msg("digest settings retrieved successfully")
	span.SetAttributes(
		attribute.Int("settings.count", len(settings)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "digest settings retrieved successfully")
	return settings, totalCount, nil
}

// DeleteDigestSettings implements BL of deleting digest settings
func (s *DigestService) DeleteDigestSettings(ctx context.Context, userID int64, messengerRelatedUserID *int) error {
	ctx, span := s.tracer.Start(ctx, "digest_service.DeleteDigestSettings",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("deleting digest settings")

	// Get existing settings to get all info for RabbitMQ message
	oldSettings, err := s.digestSettingsRepo.GetDigestSettingsByUserID(ctx, userID, messengerRelatedUserID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get digest settings for deletion")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	// Delete from repository
	err = s.digestSettingsRepo.DeleteDigestSettings(ctx, userID, messengerRelatedUserID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to delete digest settings in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	// Publish delete to RabbitMQ
	digestQueueMessage := map[string]interface{}{
		"task": "worker.delete_digest_settings",
		"args": []interface{}{
			oldSettings.UserID,
			oldSettings.MessengerRelatedUserID,
		},
	}

	err = s.producer.Publish(ctx, digestQueueMessage)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to queue delete_digest_settings message")
		// Don't fail the operation, just log the error
		// The database deletion was successful, queue update failure is non-critical
	} else {
		log.Debug().
			Int64("user.id", userID).
			Msg("delete_digest_settings message queued successfully")
	}

	log.Debug().
		Int64("user.id", userID).
		Msg("digest settings deleted successfully")
	span.SetStatus(codes.Ok, "digest settings deleted successfully")
	return nil
}

// DigestResponse represents the response for a digest
type DigestResponse struct {
	UserID                 int64          `json:"user_id"`
	MessengerRelatedUserID *int           `json:"messenger_related_user_id,omitempty"`
	ChatID                 *string        `json:"chat_id,omitempty"`
	StartDateFrom          time.Time      `json:"start_date_from"`
	StartDateTo            time.Time      `json:"start_date_to"`
	CompletedBacklogsCount int            `json:"completed_backlogs_count"`
	Tasks                  []*models.Task `json:"tasks"`
	Timezone               string         `json:"timezone"`
}

// validateTimeFormat validates that the time string is in HH:MM format
func validateTimeFormat(timeStr string) error {
	_, err := time.Parse("15:04", timeStr)
	if err != nil {
		return errors.Wrap(err, "time must be in HH:MM format (e.g., 07:00, 10:00)")
	}
	return nil
}
