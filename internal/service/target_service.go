package service

import (
	"context"

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

// TargetService defines methods for target-related business logic
type TargetService struct {
	targetRepo    repository.TargetRepository
	userRepo      repository.UserRepository
	messengerRepo repository.MessengerRepository
	tracer        trace.Tracer
	logger        zerolog.Logger
}

// NewTargetService creates a new TargetService
func NewTargetService(targetRepo repository.TargetRepository, userRepo repository.UserRepository, messengerRepo repository.MessengerRepository, logger zerolog.Logger) *TargetService {
	return &TargetService{
		targetRepo:    targetRepo,
		userRepo:      userRepo,
		messengerRepo: messengerRepo,
		tracer:        otel.Tracer("target-service"),
		logger:        logger,
	}
}

// CreateTarget implements BL of adding new target item
func (s *TargetService) CreateTarget(ctx context.Context, target *models.Target) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "target_service.CreateTarget",
		trace.WithAttributes(
			attribute.Int64("user.id", target.UserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", target.UserID).
		Str("target.title", target.Title).
		Msg("starting target creation")

	// Check if user exists
	_, err := s.userRepo.GetUserByID(ctx, target.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		log.Debug().
			Err(err).
			Int64("user.id", target.UserID).
			Msg("user not found or error retrieving user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}
	log.Debug().
		Int64("user.id", target.UserID).
		Msg("user exists, proceeding with target creation")

	if target.MessengerRelatedUserID != nil {
		span.SetAttributes(attribute.Int("messenger_related_user.id", *target.MessengerRelatedUserID))
		// Check if messenger related user exists
		_, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *target.MessengerRelatedUserID)
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
		Int64("user.id", target.UserID).
		Msg("creating target in repository")
	targetID, err := s.targetRepo.CreateTarget(ctx, target)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", target.UserID).
			Msg("failed to create target in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("target.id", targetID))
	withAuditLog(log.Debug(), buildAuditLogPayload(ctx, "created", "target", targetID, mapKeysForAudit(targetToAuditMap(target)))).
		Int64("user.id", target.UserID).
		Msg("target created successfully")

	span.SetStatus(codes.Ok, "target created successfully")
	return targetID, nil
}

// GetTargetByID implements BL of retrieving a target item by ID
func (s *TargetService) GetTargetByID(ctx context.Context, id int64) (*models.Target, error) {
	ctx, span := s.tracer.Start(ctx, "target_service.GetTargetByID",
		trace.WithAttributes(
			attribute.Int64("target.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("target.id", id).
		Msg("getting target by id")

	target, err := s.targetRepo.GetTargetByID(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("target.id", id).
			Msg("failed to get target by id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("target.id", id).
		Msg("target retrieved successfully")
	span.SetStatus(codes.Ok, "target retrieved successfully")
	return target, nil
}

// GetAllTargets implements BL of retrieving all target items with pagination, ordering, and filtering
func (s *TargetService) GetAllTargets(ctx context.Context, page, pageSize int, orderBy string, userID *int64) ([]*models.Target, int, error) {
	ctx, span := s.tracer.Start(ctx, "target_service.GetAllTargets",
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
		Msg("getting all targets")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	targets, totalCount, err := s.targetRepo.GetAllTargets(ctx, page, pageSize, orderBy, userID)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all targets")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.WithStack(err)
	}

	log.Debug().
		Int("targets.count", len(targets)).
		Int("total_count", totalCount).
		Msg("targets retrieved successfully")
	span.SetAttributes(
		attribute.Int("targets.count", len(targets)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "targets retrieved successfully")
	return targets, totalCount, nil
}

// UpdateTarget implements BL of updating a target item
func (s *TargetService) UpdateTarget(ctx context.Context, id int64, updateRequest *models.TargetUpdateRequest) (*models.Target, error) {
	ctx, span := s.tracer.Start(ctx, "target_service.UpdateTarget",
		trace.WithAttributes(
			attribute.Int64("target.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("target.id", id).
		Msg("updating target")

	// Get existing target
	oldTarget, err := s.targetRepo.GetTargetByID(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("target.id", id).
			Msg("failed to get target for update")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	beforeMap := targetToAuditMap(oldTarget)

	// Update fields if provided
	if updateRequest.Title != nil {
		if *updateRequest.Title == "" {
			err := errors.Wrap(errs.ErrValidation, "title cannot be empty")
			log.Debug().
				Err(err).
				Int64("target.id", id).
				Msg("invalid title in update request")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
		oldTarget.Title = *updateRequest.Title
	}

	if updateRequest.Description != nil {
		oldTarget.Description = *updateRequest.Description
	}

	if updateRequest.CompletedAt != nil {
		oldTarget.CompletedAt = updateRequest.CompletedAt
	}

	// Update in repository
	err = s.targetRepo.UpdateTarget(ctx, oldTarget)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("target.id", id).
			Msg("failed to update target in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	afterMap := targetToAuditMap(oldTarget)
	withAuditLog(log.Debug(), buildAuditLogPayload(ctx, "updated", "target", id, changedFieldsFromMaps(beforeMap, afterMap))).
		Int64("user.id", oldTarget.UserID).
		Msg("target updated successfully")
	span.SetStatus(codes.Ok, "target updated successfully")
	return oldTarget, nil
}

// DeleteTarget implements BL of deleting a target item
func (s *TargetService) DeleteTarget(ctx context.Context, id int64) error {
	ctx, span := s.tracer.Start(ctx, "target_service.DeleteTarget",
		trace.WithAttributes(
			attribute.Int64("target.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("target.id", id).
		Msg("deleting target")

	existingTarget, err := s.targetRepo.GetTargetByID(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("target.id", id).
			Msg("failed to get target for deletion")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	err = s.targetRepo.DeleteTarget(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("target.id", id).
			Msg("failed to delete target")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	withAuditLog(log.Debug(), buildAuditLogPayload(ctx, "deleted", "target", id, mapKeysForAudit(targetToAuditMap(existingTarget)))).
		Int64("user.id", existingTarget.UserID).
		Msg("target deleted successfully")
	span.SetStatus(codes.Ok, "target deleted successfully")
	return nil
}

func targetToAuditMap(target *models.Target) map[string]interface{} {
	result := map[string]interface{}{
		"id":          target.ID,
		"title":       target.Title,
		"description": target.Description,
		"user_id":     target.UserID,
	}

	if target.CompletedAt != nil {
		result["completed_at"] = *target.CompletedAt
	}
	if target.MessengerRelatedUserID != nil {
		result["messenger_related_user_id"] = *target.MessengerRelatedUserID
	}

	return result
}
