package mapper

import (
	"time"

	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/models"
)

// TaskHistoryModelToResponse converts models.TaskHistory to TaskHistoryResponse DTO
func TaskHistoryModelToResponse(history *models.TaskHistory) *dto.TaskHistoryResponse {
	return &dto.TaskHistoryResponse{
		ID:        history.ID,
		TaskID:    history.TaskID,
		UserID:    history.UserID,
		Action:    history.Action,
		OldValue:  history.OldValue,
		NewValue:  history.NewValue,
		CreatedAt: history.CreatedAt.Format(time.RFC3339),
	}
}

// TaskHistoriesModelToResponse converts slice of models.TaskHistory to slice of TaskHistoryResponse DTOs
func TaskHistoriesModelToResponse(histories []*models.TaskHistory) []*dto.TaskHistoryResponse {
	responses := make([]*dto.TaskHistoryResponse, len(histories))
	for i, history := range histories {
		responses[i] = TaskHistoryModelToResponse(history)
	}
	return responses
}
