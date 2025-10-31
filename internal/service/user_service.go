package service

import (
	"context"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/queue"

	"github.com/pkg/errors"
)

// UserService defines methods for user-related business logic
type UserService struct {
	userRepo      repository.UserRepository
	taskRepo      repository.TaskRepository
	messengerRepo repository.MessengerRepository
	producer      *queue.Producer
}

// NewUserService creates a new instance of UserService
func NewUserService(userRepo repository.UserRepository, taskRepo repository.TaskRepository, messengerRepo repository.MessengerRepository, producer *queue.Producer) *UserService {
	return &UserService{userRepo: userRepo, taskRepo: taskRepo, messengerRepo: messengerRepo, producer: producer}
}

// CreateUser implements BL of adding new user
func (s *UserService) CreateUser(ctx context.Context, user *models.User) (int64, error) {
	// perform some validation before creating the user
	if user.Name == "" {
		return 0, errors.WithStack(errors.New("user data is incomplete"))
	}

	userID, err := s.userRepo.CreateUser(ctx, user)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return userID, nil
}

// GetUser implements BL of retrieving existing user by its id
func (s *UserService) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return user, nil
}

// UpdateUser implements BL of updating user by id
func (s *UserService) UpdateUser(ctx context.Context, userID int64, updateRequest *models.UserUpdateRequest) (*models.User, error) {
	// check if the user exists
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// update the user fields (partial update)
	if updateRequest.Name != nil {
		user.Name = *updateRequest.Name
	}
	if updateRequest.Email != nil {
		user.Email = *updateRequest.Email
	}
	if updateRequest.PasswordHash != nil {
		user.PasswordHash = *updateRequest.PasswordHash
	}
	if updateRequest.Timezone != nil {
		user.Timezone = updateRequest.Timezone
	}
	if updateRequest.LanguageCode != nil {
		user.LanguageCode = updateRequest.LanguageCode
	}
	if updateRequest.Role != nil {
		user.Role = updateRequest.Role
	}

	// save the updated user
	err = s.userRepo.UpdateUser(ctx, user)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return user, nil
}

// DeleteUser implements BL of soft deleting user by id
func (s *UserService) DeleteUser(ctx context.Context, userID int64) error {
	_, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return errors.WithStack(err)
	}

	tasks, err := s.taskRepo.GetTasksByUserID(ctx, userID)
	if err != nil {
		// TODO: hhtp code?
		return errors.WithStack(err)
	}
	for _, task := range tasks {
		// TODO: allow validation + check errors
		err = s.taskRepo.DeleteTask(ctx, task.ID)
		if err != nil {
			// retry or rollback
		}

		taskQueueMessage := map[string]interface{}{
			"task": "worker.delete_task",
			"args": []interface{}{task.ID, "telegram"},
		}

		err = s.producer.Publish(ctx, taskQueueMessage)
		if err != nil {
			// TODO: failed to publish message: Exception (504) Reason: \"channel/connection is not open\"
			return errors.WithStack(errors.Errorf("can't publish message %v to rabbitmq: %s",
				taskQueueMessage,
				err,
			))

		}
	}

	err = s.messengerRepo.DeleteMessengerRelatedUserByUserID(ctx, userID)
	if err != nil {
		return errors.WithStack(err)
	}

	err = s.userRepo.DeleteUser(ctx, userID)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
