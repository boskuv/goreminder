package service

import (
	"context"
	"strings"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// BacklogService defines methods for backlog-related business logic
type BacklogService struct {
	backlogRepo   repository.BacklogRepository
	userRepo      repository.UserRepository
	messengerRepo repository.MessengerRepository
	tracer        trace.Tracer
	logger        zerolog.Logger
}

// NewBacklogService creates a new BacklogService
func NewBacklogService(backlogRepo repository.BacklogRepository, userRepo repository.UserRepository, messengerRepo repository.MessengerRepository, logger zerolog.Logger) *BacklogService {
	return &BacklogService{
		backlogRepo:   backlogRepo,
		userRepo:      userRepo,
		messengerRepo: messengerRepo,
		tracer:        otel.Tracer("backlog-service"),
		logger:        logger,
	}
}

// CreateBacklog implements BL of adding new backlog item
func (s *BacklogService) CreateBacklog(ctx context.Context, backlog *models.Backlog) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "backlog_service.CreateBacklog",
		trace.WithAttributes(
			attribute.Int64("user.id", backlog.UserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", backlog.UserID).
		Str("backlog.title", backlog.Title).
		Msg("starting backlog creation")

	// Check if user exists
	_, err := s.userRepo.GetUserByID(ctx, backlog.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		log.Debug().
			Err(err).
			Int64("user.id", backlog.UserID).
			Msg("user not found or error retrieving user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}
	log.Debug().
		Int64("user.id", backlog.UserID).
		Msg("user exists, proceeding with backlog creation")

	if backlog.MessengerRelatedUserID != nil {
		span.SetAttributes(attribute.Int("messenger_related_user.id", *backlog.MessengerRelatedUserID))
		// Check if messenger related user exists
		_, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *backlog.MessengerRelatedUserID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return 0, errors.WithStack(err)
		}
	}

	log.Debug().
		Int64("user.id", backlog.UserID).
		Msg("creating backlog in repository")
	backlogID, err := s.backlogRepo.CreateBacklog(ctx, backlog)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", backlog.UserID).
			Msg("failed to create backlog in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("backlog.id", backlogID))
	log.Debug().
		Int64("backlog.id", backlogID).
		Int64("user.id", backlog.UserID).
		Msg("backlog created successfully")

	span.SetStatus(codes.Ok, "backlog created successfully")
	return backlogID, nil
}

// CreateBacklogsBatch implements BL of adding multiple backlog items from a batch string
// Items are separated by separator (default: newline)
func (s *BacklogService) CreateBacklogsBatch(ctx context.Context, items string, separator string, userID int64, messengerRelatedUserID *int) ([]int64, error) {
	ctx, span := s.tracer.Start(ctx, "backlog_service.CreateBacklogsBatch",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("starting batch backlog creation")

	// Check if user exists
	_, err := s.userRepo.GetUserByID(ctx, userID)
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

	if messengerRelatedUserID != nil {
		span.SetAttributes(attribute.Int("messenger_related_user.id", *messengerRelatedUserID))
		// Check if messenger related user exists
		_, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *messengerRelatedUserID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
	}

	// Default separator is newline if not provided
	if separator == "" {
		separator = "\n"
	}

	// Split items by separator
	itemLines := strings.Split(items, separator)
	var createdIDs []int64

	for _, line := range itemLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue // Skip empty lines
		}

		backlog := &models.Backlog{
			Title:                  line,
			Description:            "",
			UserID:                 userID,
			MessengerRelatedUserID: messengerRelatedUserID,
		}

		backlogID, err := s.backlogRepo.CreateBacklog(ctx, backlog)
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Int64("user.id", userID).
				Str("item", line).
				Msg("failed to create backlog item in batch")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			// Continue with other items, but log the error
			continue
		}

		createdIDs = append(createdIDs, backlogID)
		log.Debug().
			Int64("backlog.id", backlogID).
			Str("item", line).
			Msg("backlog item created in batch")
	}

	if len(createdIDs) == 0 {
		err := errors.Wrap(errs.ErrUnprocessableEntity, "no valid items to create")
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("no valid items found in batch")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int("created.count", len(createdIDs)))
	log.Debug().
		Int64("user.id", userID).
		Int("created.count", len(createdIDs)).
		Msg("batch backlog creation completed successfully")

	span.SetStatus(codes.Ok, "batch backlog creation completed successfully")
	return createdIDs, nil
}

// GetBacklogByID implements BL of retrieving a backlog item by ID
func (s *BacklogService) GetBacklogByID(ctx context.Context, id int64) (*models.Backlog, error) {
	ctx, span := s.tracer.Start(ctx, "backlog_service.GetBacklogByID",
		trace.WithAttributes(
			attribute.Int64("backlog.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("backlog.id", id).
		Msg("getting backlog by id")

	backlog, err := s.backlogRepo.GetBacklogByID(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("backlog.id", id).
			Msg("failed to get backlog by id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("backlog.id", id).
		Msg("backlog retrieved successfully")
	span.SetStatus(codes.Ok, "backlog retrieved successfully")
	return backlog, nil
}

// GetAllBacklogs implements BL of retrieving all backlog items with pagination, ordering, and filtering
func (s *BacklogService) GetAllBacklogs(ctx context.Context, page, pageSize int, orderBy string, userID *int64, completed *bool) ([]*models.Backlog, int, error) {
	ctx, span := s.tracer.Start(ctx, "backlog_service.GetAllBacklogs",
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
		Msg("getting all backlogs")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	backlogs, totalCount, err := s.backlogRepo.GetAllBacklogs(ctx, page, pageSize, orderBy, userID, completed)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all backlogs")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.WithStack(err)
	}

	log.Debug().
		Int("backlogs.count", len(backlogs)).
		Int("total_count", totalCount).
		Msg("backlogs retrieved successfully")
	span.SetAttributes(
		attribute.Int("backlogs.count", len(backlogs)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "backlogs retrieved successfully")
	return backlogs, totalCount, nil
}

// UpdateBacklog implements BL of updating a backlog item
func (s *BacklogService) UpdateBacklog(ctx context.Context, id int64, updateRequest *models.BacklogUpdateRequest) (*models.Backlog, error) {
	ctx, span := s.tracer.Start(ctx, "backlog_service.UpdateBacklog",
		trace.WithAttributes(
			attribute.Int64("backlog.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("backlog.id", id).
		Msg("updating backlog")

	// Get existing backlog
	oldBacklog, err := s.backlogRepo.GetBacklogByID(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("backlog.id", id).
			Msg("failed to get backlog for update")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Update fields if provided
	if updateRequest.Title != nil {
		if *updateRequest.Title == "" {
			err := errors.Wrap(errs.ErrValidation, "title cannot be empty")
			log.Debug().
				Err(err).
				Int64("backlog.id", id).
				Msg("invalid title in update request")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
		oldBacklog.Title = *updateRequest.Title
	}

	if updateRequest.Description != nil {
		oldBacklog.Description = *updateRequest.Description
	}

	if updateRequest.CompletedAt != nil {
		oldBacklog.CompletedAt = updateRequest.CompletedAt
	}

	// Update in repository
	err = s.backlogRepo.UpdateBacklog(ctx, oldBacklog)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("backlog.id", id).
			Msg("failed to update backlog in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("backlog.id", id).
		Msg("backlog updated successfully")
	span.SetStatus(codes.Ok, "backlog updated successfully")
	return oldBacklog, nil
}

// DeleteBacklog implements BL of deleting a backlog item
func (s *BacklogService) DeleteBacklog(ctx context.Context, id int64) error {
	ctx, span := s.tracer.Start(ctx, "backlog_service.DeleteBacklog",
		trace.WithAttributes(
			attribute.Int64("backlog.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("backlog.id", id).
		Msg("deleting backlog")

	err := s.backlogRepo.DeleteBacklog(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("backlog.id", id).
			Msg("failed to delete backlog")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	log.Debug().
		Int64("backlog.id", id).
		Msg("backlog deleted successfully")
	span.SetStatus(codes.Ok, "backlog deleted successfully")
	return nil
}
