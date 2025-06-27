package service

import (
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"

	"github.com/pkg/errors"
)

// UserService defines methods for user-related business logic
type UserService struct {
	userRepo repository.UserRepository
}

// NewUserService creates a new instance of UserService
func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// CreateUser implements BL of adding new user
func (s *UserService) CreateUser(user *models.User) (int64, error) {
	// perform some validation before creating the user
	if user.Name == "" {
		return 0, errors.WithStack(errors.New("user data is incomplete"))
	}

	userID, err := s.userRepo.CreateUser(user)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return userID, nil
}

// GetUser implements BL of retrieving existing user by its id
func (s *UserService) GetUser(userID int64) (*models.User, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return user, nil
}

// UpdateUser implements BL of updating user by id
func (s *UserService) UpdateUser(userID int64, updateRequest *models.UserUpdateRequest) (*models.User, error) {
	// check if the user exists
	user, err := s.userRepo.GetUserByID(userID)
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

	// save the updated user
	err = s.userRepo.UpdateUser(user)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return user, nil
}

// DeleteUser implements BL of soft deleting user by id
func (s *UserService) DeleteUser(userID int64) error {
	_, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return errors.WithStack(err)
	}

	err = s.userRepo.DeleteUser(userID)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
