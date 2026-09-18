package handlers

import (
	"context"

	"github.com/boskuv/goreminder/internal/models"
)

// UserService defines the user use-case surface used by UserHandler.
// *service.UserService implements this interface; tests can substitute a stub.
type UserService interface {
	CreateUser(ctx context.Context, user *models.User) (int64, error)
	GetUser(ctx context.Context, userID int64) (*models.User, error)
	UpdateUser(ctx context.Context, userID int64, updateRequest *models.UserUpdateRequest) (*models.User, error)
	DeleteUser(ctx context.Context, userID int64) error
	GetAllUsers(ctx context.Context, page, pageSize int, orderBy string) ([]*models.User, int, error)
	GetRecentUserActivity(ctx context.Context, limit int) ([]models.UserActivity, error)
}
