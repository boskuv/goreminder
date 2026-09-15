package handlers

import (
	"context"
	"time"

	"github.com/boskuv/goreminder/internal/models"
)

// TaskService defines the task use-case surface used by TaskHandler.
// *service.TaskService implements this interface; tests can substitute a stub.
type TaskService interface {
	CreateTask(ctx context.Context, task *models.Task) (int64, int64, error)
	GetTask(ctx context.Context, taskID int64) (*models.Task, error)
	GetUserTasks(ctx context.Context, userID int64, page, pageSize int, orderBy string, startDateFrom, startDateTo, createdAtFrom, createdAtTo *time.Time, requiresConfirmation *bool, status *string, statusNot *string, cronExpression *string, cronExpressionIsNull *bool, excludeCronWithConfirmation *bool, messengerRelatedUserID *int, messengerUserID *string) ([]*models.Task, int, error)
	UpdateTask(ctx context.Context, taskID int64, updateRequest *models.TaskUpdateRequest) (*models.Task, error)
	DeleteTask(ctx context.Context, taskID int64) error
	QueueTask(ctx context.Context, scheduledTask *models.ScheduledTask) error
	MarkTaskAsDone(ctx context.Context, taskID int64) (*models.Task, error)
	MuteTask(ctx context.Context, taskID int64) (*models.Task, error)
	UnmuteTask(ctx context.Context, taskID int64) (*models.Task, error)
	GetTaskHistory(ctx context.Context, taskID int64) ([]*models.TaskHistory, error)
	GetUserTaskHistory(ctx context.Context, userID int64, limit, offset int) ([]*models.TaskHistory, error)
	GetAllTasks(ctx context.Context, page, pageSize int, orderBy string, status *string, statusNot *string, startDateFrom *time.Time, startDateTo *time.Time, userID *int64, cronExpression *string, cronExpressionIsNull *bool, requiresConfirmation *bool, excludeCronWithConfirmation *bool) ([]*models.Task, int, error)
}
