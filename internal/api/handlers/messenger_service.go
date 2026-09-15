package handlers

import (
	"context"

	"github.com/boskuv/goreminder/internal/models"
)

// MessengerService defines the messenger use-case surface used by MessengerHandler.
// *service.MessengerService implements this interface; tests can substitute a stub.
type MessengerService interface {
	CreateMessenger(ctx context.Context, messenger *models.Messenger) (int64, error)
	GetMessenger(ctx context.Context, messengerID int64) (*models.Messenger, error)
	GetMessengerIDByName(ctx context.Context, messengerName string) (int64, error)
	CreateMessengerRelatedUser(ctx context.Context, messengerRelatedUser *models.MessengerRelatedUser) (int64, error)
	GetMessengerRelatedUser(ctx context.Context, chatID string, messengerUserID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error)
	GetUserID(ctx context.Context, messengerUserID string) (int64, error)
	GetAllMessengers(ctx context.Context, page, pageSize int, orderBy string) ([]*models.Messenger, int, error)
	GetAllMessengerRelatedUsers(ctx context.Context, page, pageSize int, orderBy string, userID *int64, chatID *string) ([]*models.MessengerRelatedUser, int, error)
}
