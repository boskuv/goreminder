package service

import (
	"context"

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

// UserService defines methods for user-related business logic
type UserService struct {
	userRepo      repository.UserRepository
	taskRepo      repository.TaskRepository
	messengerRepo repository.MessengerRepository
	producer      queue.Publisher
	tracer        trace.Tracer
	logger        zerolog.Logger
}

// NewUserService creates a new instance of UserService
func NewUserService(userRepo repository.UserRepository, taskRepo repository.TaskRepository, messengerRepo repository.MessengerRepository, producer queue.Publisher, logger zerolog.Logger) *UserService {
	return &UserService{
		userRepo:      userRepo,
		taskRepo:      taskRepo,
		messengerRepo: messengerRepo,
		producer:      producer,
		tracer:        otel.Tracer("user-service"),
		logger:        logger,
	}
}

// getMessengerName gets messenger name from MessengerRelatedUser
// Returns messenger name or empty string if not found
func (s *UserService) getMessengerName(ctx context.Context, messengerRelatedUserID int) (string, error) {
	messengerRelatedUser, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, messengerRelatedUserID)
	if err != nil {
		return "", errors.WithStack(err)
	}

	if messengerRelatedUser.MessengerID == nil {
		return "", errors.Wrap(errs.ErrUnprocessableEntity, "messenger_id is nil in messenger_related_user")
	}

	messenger, err := s.messengerRepo.GetMessengerByID(ctx, *messengerRelatedUser.MessengerID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return "", errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		return "", errors.WithStack(err)
	}

	return messenger.Name, nil
}

// CreateUser implements BL of adding new user
func (s *UserService) CreateUser(ctx context.Context, user *models.User) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "user_service.CreateUser",
		trace.WithAttributes(
			attribute.String("user.name", user.Name),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Str("user.name", user.Name).
		Str("user.email", user.Email).
		Msg("starting user creation")

	// perform some validation before creating the user
	if user.Name == "" {
		err := errors.New("user data is incomplete")
		log.Debug().
			Err(err).
			Msg("user validation failed: name is empty")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	log.Debug().
		Str("user.name", user.Name).
		Msg("creating user in repository")
	userID, err := s.userRepo.CreateUser(ctx, user)
	if err != nil {
		log.Debug().
			Err(err).
			Str("user.name", user.Name).
			Msg("failed to create user in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("user.id", userID))
	log.Debug().
		Int64("user.id", userID).
		Str("user.name", user.Name).
		Msg("user created successfully")
	span.SetStatus(codes.Ok, "user created successfully")
	return userID, nil
}

// GetUser implements BL of retrieving existing user by its id
func (s *UserService) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	ctx, span := s.tracer.Start(ctx, "user_service.GetUser",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("getting user")

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("user.id", userID).
		Msg("user retrieved successfully")
	span.SetStatus(codes.Ok, "user retrieved successfully")
	return user, nil
}

// UpdateUser implements BL of updating user by id
func (s *UserService) UpdateUser(ctx context.Context, userID int64, updateRequest *models.UserUpdateRequest) (*models.User, error) {
	ctx, span := s.tracer.Start(ctx, "user_service.UpdateUser",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("updating user")

	// check if the user exists
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get user for update")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// update the user fields (partial update)
	if updateRequest.Name != nil {
		user.Name = *updateRequest.Name
		span.SetAttributes(attribute.String("user.name.updated", user.Name))
	}
	if updateRequest.Email != nil {
		user.Email = *updateRequest.Email
		span.SetAttributes(attribute.String("user.email.updated", user.Email))
	}
	if updateRequest.PasswordHash != nil {
		user.PasswordHash = *updateRequest.PasswordHash
		span.SetAttributes(attribute.Bool("user.password.updated", true))
	}
	if updateRequest.Timezone != nil {
		user.Timezone = updateRequest.Timezone
		if *user.Timezone != "" {
			span.SetAttributes(attribute.String("user.timezone.updated", *user.Timezone))
		}
	}
	if updateRequest.LanguageCode != nil {
		user.LanguageCode = updateRequest.LanguageCode
		if *user.LanguageCode != "" {
			span.SetAttributes(attribute.String("user.language_code.updated", *user.LanguageCode))
		}
	}
	if updateRequest.Role != nil {
		user.Role = updateRequest.Role
		if *user.Role != "" {
			span.SetAttributes(attribute.String("user.role.updated", *user.Role))
		}
	}

	// save the updated user
	err = s.userRepo.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to update user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("user.id", userID).
		Msg("user updated successfully")
	span.SetStatus(codes.Ok, "user updated successfully")
	return user, nil
}

// DeleteUser implements BL of soft deleting user by id
func (s *UserService) DeleteUser(ctx context.Context, userID int64) error {
	ctx, span := s.tracer.Start(ctx, "user_service.DeleteUser",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", userID).
		Msg("deleting user")

	_, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get user for deletion")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	tasks, err := s.taskRepo.GetTasksByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int("tasks.count", len(tasks)))
	for _, task := range tasks {
		err = s.taskRepo.DeleteTask(ctx, task.ID)
		if err != nil {
			span.RecordError(err)
			// retry or rollback
		}

		var messengerName string
		if task.MessengerRelatedUserID != nil {
			messengerName, err = s.getMessengerName(ctx, int(*task.MessengerRelatedUserID))
			if err != nil {
				log.Error().
					Stack().
					Err(err).
					Int64("task.id", task.ID).
					Msg("failed to get messenger name for delete_task")
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				// Continue with next task
				continue
			}
		} else {
			log.Error().
				Int64("task.id", task.ID).
				Msg("task has no messenger_related_user_id")
			// Continue with next task
			continue
		}

		taskQueueMessage := map[string]interface{}{
			"task": "worker.delete_task",
			"args": []interface{}{task.ID, messengerName},
		}

		err = s.producer.Publish(ctx, taskQueueMessage)
		if err != nil {
			err = errors.Errorf("can't publish message %v to rabbitmq: %s",
				taskQueueMessage,
				err,
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return errors.WithStack(err)
		}
	}

	err = s.messengerRepo.DeleteMessengerRelatedUserByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	err = s.userRepo.DeleteUser(ctx, userID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to delete user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	log.Debug().
		Int64("user.id", userID).
		Msg("user deleted successfully")
	span.SetStatus(codes.Ok, "user deleted successfully")
	return nil
}

// GetAllUsers implements BL of retrieving all users with pagination and ordering
func (s *UserService) GetAllUsers(ctx context.Context, page, pageSize int, orderBy string) ([]*models.User, int, error) {
	ctx, span := s.tracer.Start(ctx, "user_service.GetAllUsers",
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
		Msg("getting all users")

	users, totalCount, err := s.userRepo.GetAllUsers(ctx, page, pageSize, orderBy)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all users")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.WithStack(err)
	}

	log.Debug().
		Int("users.count", len(users)).
		Int("total_count", totalCount).
		Msg("users retrieved successfully")
	span.SetAttributes(
		attribute.Int("users.count", len(users)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "users retrieved successfully")
	return users, totalCount, nil
}
