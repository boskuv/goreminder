package handlers

import (
	"context"

	"github.com/boskuv/goreminder/internal/models"
)

// BacklogService defines the backlog use-case surface used by BacklogHandler.
// *service.BacklogService implements this interface; tests can substitute a stub.
type BacklogService interface {
	CreateBacklog(ctx context.Context, backlog *models.Backlog) (int64, error)
	CreateBacklogsBatch(ctx context.Context, items string, separator string, userID int64, messengerRelatedUserID *int) ([]int64, error)
	GetBacklogByID(ctx context.Context, id int64) (*models.Backlog, error)
	GetAllBacklogs(ctx context.Context, page, pageSize int, orderBy string, userID *int64, completed *bool, messengerUserID *string) ([]*models.Backlog, int, error)
	UpdateBacklog(ctx context.Context, id int64, updateRequest *models.BacklogUpdateRequest) (*models.Backlog, error)
	DeleteBacklog(ctx context.Context, id int64) error
}
