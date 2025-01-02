package service

import (
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"

	"github.com/pkg/errors"
)

// UserService handles user-related business logic.
type UserService struct {
	userRepo repository.UserRepository
}

// NewUserService creates a new instance of UserService.
func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// CreateUser creates a new user in the system.
func (s *UserService) CreateUser(user *models.User) (int64, error) {
	// Perform some validation before creating the user
	if user.Name == "" || user.Email == "" || user.PasswordHash == "" {
		return 0, errors.WithStack(errors.New("user data is incomplete"))
	}

	// Call the repository to insert the user into the database
	userID, err := s.userRepo.CreateUser(user)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

// GetUser retrieves a user by its ID
func (s *UserService) GetUser(userID int64) (*models.User, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteUser deletes a user by its ID (soft delete)
func (s *UserService) DeleteUser(userID int64) error {
	_, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return err
	}

	err = s.userRepo.DeleteUser(userID)
	if err != nil {
		return err
	}

	return nil
}
