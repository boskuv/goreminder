package service

import (
	"context"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/queue"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pkg/errors"
)

// UserService defines methods for user-related business logic
type UserService struct {
	userRepo      repository.UserRepository
	taskRepo      repository.TaskRepository
	messengerRepo repository.MessengerRepository
	producer      *queue.Producer
	tracer        trace.Tracer
}

// NewUserService creates a new instance of UserService
func NewUserService(userRepo repository.UserRepository, taskRepo repository.TaskRepository, messengerRepo repository.MessengerRepository, producer *queue.Producer) *UserService {
	return &UserService{
		userRepo:      userRepo,
		taskRepo:      taskRepo,
		messengerRepo: messengerRepo,
		producer:      producer,
		tracer:        otel.Tracer("user-service"),
	}
}

// CreateUser implements BL of adding new user
func (s *UserService) CreateUser(ctx context.Context, user *models.User) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "user_service.CreateUser",
		trace.WithAttributes(
			attribute.String("user.name", user.Name),
		))
	defer span.End()

	// perform some validation before creating the user
	if user.Name == "" {
		err := errors.New("user data is incomplete")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	userID, err := s.userRepo.CreateUser(ctx, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("user.id", userID))
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

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

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

	// check if the user exists
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

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

	_, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
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
		// TODO: allow validation + check errors
		err = s.taskRepo.DeleteTask(ctx, task.ID)
		if err != nil {
			span.RecordError(err)
			// retry or rollback
		}

		taskQueueMessage := map[string]interface{}{
			"task": "worker.delete_task",
			"args": []interface{}{task.ID, "telegram"},
		}

		err = s.producer.Publish(ctx, taskQueueMessage)
		if err != nil {
			// TODO: failed to publish message: Exception (504) Reason: \"channel/connection is not open\"
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	span.SetStatus(codes.Ok, "user deleted successfully")
	return nil
}
