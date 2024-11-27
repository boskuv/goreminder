package service

import (
	"errors"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
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
		return 0, errors.New("user data is incomplete")
	}

	// Call the repository to insert the user into the database
	userID, err := s.userRepo.CreateUser(user)
	if err != nil {
		return 0, err
	}

	return userID, nil
}
