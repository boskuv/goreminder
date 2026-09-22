package handlers

import (
	"context"

	"github.com/boskuv/goreminder/internal/models"
)

// TaskGroupService defines the task-group use-case surface used by TaskGroupHandler.
// *service.TaskGroupService implements this interface; tests can substitute a stub.
type TaskGroupService interface {
	CreateTaskGroup(ctx context.Context, group *models.TaskGroup) (int64, error)
	GetTaskGroupByID(ctx context.Context, id int64) (*models.TaskGroup, error)
	GetAllTaskGroups(ctx context.Context, page, pageSize int, orderBy string, userID *int64) ([]*models.TaskGroup, int, error)
	UpdateTaskGroup(ctx context.Context, id int64, updateRequest *models.TaskGroupUpdateRequest) (*models.TaskGroup, error)
	DeleteTaskGroup(ctx context.Context, id int64) error
}
