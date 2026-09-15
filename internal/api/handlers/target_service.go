package handlers

import (
	"context"

	"github.com/boskuv/goreminder/internal/models"
)

// TargetService defines the target use-case surface used by TargetHandler.
// *service.TargetService implements this interface; tests can substitute a stub.
type TargetService interface {
	CreateTarget(ctx context.Context, target *models.Target) (int64, error)
	GetTargetByID(ctx context.Context, id int64) (*models.Target, error)
	GetAllTargets(ctx context.Context, page, pageSize int, orderBy string, userID *int64, messengerUserID *string) ([]*models.Target, int, error)
	UpdateTarget(ctx context.Context, id int64, updateRequest *models.TargetUpdateRequest) (*models.Target, error)
	DeleteTarget(ctx context.Context, id int64) error
}
