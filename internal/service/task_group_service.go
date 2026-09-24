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

// TaskGroupService defines methods for task-group-related business logic
type TaskGroupService struct {
	taskGroupRepo repository.TaskGroupRepository
	userRepo      repository.UserRepository
	bindings      repository.CalendarBindingRepository // optional; nil skips calendar binding checks
	activity      ActivityTracker
	tracer        trace.Tracer
	logger        zerolog.Logger
}

// NewTaskGroupService creates a new TaskGroupService.
// Pass a non-nil bindings repo to block deleting groups that are referenced by calendar bindings.
func NewTaskGroupService(
	taskGroupRepo repository.TaskGroupRepository,
	userRepo repository.UserRepository,
	activity ActivityTracker,
	logger zerolog.Logger,
	bindings repository.CalendarBindingRepository,
) *TaskGroupService {
	if activity == nil {
		activity = NoopActivityTracker{}
	}
	return &TaskGroupService{
		taskGroupRepo: taskGroupRepo,
		userRepo:      userRepo,
		bindings:      bindings,
		activity:      activity,
		tracer:        otel.Tracer("task-group-service"),
		logger:        logger,
	}
}

// CreateTaskGroup implements BL of adding a new task group
func (s *TaskGroupService) CreateTaskGroup(ctx context.Context, group *models.TaskGroup) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "task_group_service.CreateTaskGroup",
		trace.WithAttributes(
			attribute.Int64("user.id", group.UserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", group.UserID).
		Str("task_group.name", group.Name).
		Msg("starting task group creation")

	if group.Name == "" {
		err := errors.Wrap(errs.ErrValidation, "name cannot be empty")
		log.Debug().Err(err).Msg("invalid task group name")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	_, err := s.userRepo.GetUserByID(ctx, group.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		log.Debug().
			Err(err).
			Int64("user.id", group.UserID).
			Msg("user not found or error retrieving user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}
	log.Debug().
		Int64("user.id", group.UserID).
		Msg("user exists, proceeding with task group creation")

	groupID, err := s.taskGroupRepo.CreateTaskGroup(ctx, group)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", group.UserID).
			Msg("failed to create task group in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("task_group.id", groupID))
	withAuditLog(log.Debug(), buildAuditLogPayload(ctx, "created", "task_group", groupID, mapKeysForAudit(taskGroupToAuditMap(group)))).
		Int64("user.id", group.UserID).
		Msg("task group created successfully")

	touchUserActivity(ctx, s.activity, s.logger, group.UserID)
	span.SetStatus(codes.Ok, "task group created successfully")
	return groupID, nil
}

// GetTaskGroupByID implements BL of retrieving a task group by ID
func (s *TaskGroupService) GetTaskGroupByID(ctx context.Context, id int64) (*models.TaskGroup, error) {
	ctx, span := s.tracer.Start(ctx, "task_group_service.GetTaskGroupByID",
		trace.WithAttributes(
			attribute.Int64("task_group.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("task_group.id", id).
		Msg("getting task group by id")

	group, err := s.taskGroupRepo.GetTaskGroupByID(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task_group.id", id).
			Msg("failed to get task group by id")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("task_group.id", id).
		Msg("task group retrieved successfully")
	span.SetStatus(codes.Ok, "task group retrieved successfully")
	return group, nil
}

// GetAllTaskGroups implements BL of retrieving task groups with pagination and filtering
func (s *TaskGroupService) GetAllTaskGroups(ctx context.Context, page, pageSize int, orderBy string, userID *int64) ([]*models.TaskGroup, int, error) {
	ctx, span := s.tracer.Start(ctx, "task_group_service.GetAllTaskGroups",
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
		Msg("getting all task groups")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	groups, totalCount, err := s.taskGroupRepo.GetAllTaskGroups(ctx, page, pageSize, orderBy, userID)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all task groups")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.WithStack(err)
	}

	log.Debug().
		Int("task_groups.count", len(groups)).
		Int("total_count", totalCount).
		Msg("task groups retrieved successfully")
	span.SetAttributes(
		attribute.Int("task_groups.count", len(groups)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "task groups retrieved successfully")
	return groups, totalCount, nil
}

// UpdateTaskGroup implements BL of updating a task group
func (s *TaskGroupService) UpdateTaskGroup(ctx context.Context, id int64, updateRequest *models.TaskGroupUpdateRequest) (*models.TaskGroup, error) {
	ctx, span := s.tracer.Start(ctx, "task_group_service.UpdateTaskGroup",
		trace.WithAttributes(
			attribute.Int64("task_group.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("task_group.id", id).
		Msg("updating task group")

	oldGroup, err := s.taskGroupRepo.GetTaskGroupByID(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task_group.id", id).
			Msg("failed to get task group for update")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	beforeMap := taskGroupToAuditMap(oldGroup)

	if updateRequest.Name != nil {
		if *updateRequest.Name == "" {
			err := errors.Wrap(errs.ErrValidation, "name cannot be empty")
			log.Debug().
				Err(err).
				Int64("task_group.id", id).
				Msg("invalid name in update request")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}
		oldGroup.Name = *updateRequest.Name
	}

	err = s.taskGroupRepo.UpdateTaskGroup(ctx, oldGroup)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task_group.id", id).
			Msg("failed to update task group in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	afterMap := taskGroupToAuditMap(oldGroup)
	withAuditLog(log.Debug(), buildAuditLogPayload(ctx, "updated", "task_group", id, changedFieldsFromMaps(beforeMap, afterMap))).
		Int64("user.id", oldGroup.UserID).
		Msg("task group updated successfully")
	touchUserActivity(ctx, s.activity, s.logger, oldGroup.UserID)
	span.SetStatus(codes.Ok, "task group updated successfully")
	return oldGroup, nil
}

// DeleteTaskGroup implements BL of soft-deleting a task group.
// Deletion is blocked (conflict) while any non-deleted calendar binding references the group.
func (s *TaskGroupService) DeleteTaskGroup(ctx context.Context, id int64) error {
	ctx, span := s.tracer.Start(ctx, "task_group_service.DeleteTaskGroup",
		trace.WithAttributes(
			attribute.Int64("task_group.id", id),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("task_group.id", id).
		Msg("deleting task group")

	existingGroup, err := s.taskGroupRepo.GetTaskGroupByID(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task_group.id", id).
			Msg("failed to get task group for deletion")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	if s.bindings != nil {
		n, err := s.bindings.CountByGroupID(ctx, id)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return errors.WithStack(err)
		}
		if n > 0 {
			err := errors.Wrap(errs.ErrConflict,
				"cannot delete task group while calendar bindings reference it; delete or reassign those bindings first")
			log.Info().
				Err(err).
				Int64("task_group.id", id).
				Int("calendar_bindings.count", n).
				Msg("task group delete blocked by calendar bindings")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	err = s.taskGroupRepo.DeleteTaskGroup(ctx, id)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task_group.id", id).
			Msg("failed to delete task group")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	withAuditLog(log.Debug(), buildAuditLogPayload(ctx, "deleted", "task_group", id, mapKeysForAudit(taskGroupToAuditMap(existingGroup)))).
		Int64("user.id", existingGroup.UserID).
		Msg("task group deleted successfully")
	touchUserActivity(ctx, s.activity, s.logger, existingGroup.UserID)
	span.SetStatus(codes.Ok, "task group deleted successfully")
	return nil
}

func taskGroupToAuditMap(group *models.TaskGroup) map[string]interface{} {
	return map[string]interface{}{
		"id":      group.ID,
		"user_id": group.UserID,
		"name":    group.Name,
	}
}
