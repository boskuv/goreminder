package mapper

import (
	"time"

	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/service"
)

// CreateDigestSettingsRequestToModel converts CreateDigestSettingsRequest DTO to models.DigestSettings
func CreateDigestSettingsRequestToModel(req *dto.CreateDigestSettingsRequest) *models.DigestSettings {
	return &models.DigestSettings{
		UserID:                 req.UserID,
		MessengerRelatedUserID: req.MessengerRelatedUserID,
		Enabled:                req.Enabled,
		WeekdayTime:            req.WeekdayTime,
		WeekendTime:            req.WeekendTime,
	}
}

// UpdateDigestSettingsRequestToModel converts UpdateDigestSettingsRequest DTO to models.DigestSettingsUpdateRequest
func UpdateDigestSettingsRequestToModel(req *dto.UpdateDigestSettingsRequest) *models.DigestSettingsUpdateRequest {
	return &models.DigestSettingsUpdateRequest{
		Enabled:                req.Enabled,
		WeekdayTime:            req.WeekdayTime,
		WeekendTime:            req.WeekendTime,
		MessengerRelatedUserID: req.MessengerRelatedUserID,
	}
}

// DigestSettingsModelToResponse converts models.DigestSettings to DigestSettingsResponse DTO
func DigestSettingsModelToResponse(settings *models.DigestSettings) *dto.DigestSettingsResponse {
	return &dto.DigestSettingsResponse{
		ID:                     settings.ID,
		UserID:                 settings.UserID,
		MessengerRelatedUserID: settings.MessengerRelatedUserID,
		Enabled:                settings.Enabled,
		WeekdayTime:            settings.WeekdayTime,
		WeekendTime:            settings.WeekendTime,
		CreatedAt:              settings.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              settings.UpdatedAt.Format(time.RFC3339),
	}
}

// DigestSettingsModelsToResponse converts a slice of models.DigestSettings to a slice of DigestSettingsResponse DTOs
func DigestSettingsModelsToResponse(settings []*models.DigestSettings) []dto.DigestSettingsResponse {
	responses := make([]dto.DigestSettingsResponse, len(settings))
	for i, s := range settings {
		response := DigestSettingsModelToResponse(s)
		responses[i] = *response
	}
	return responses
}

// DigestServiceResponseToDTO converts service.DigestResponse to dto.DigestResponse
// Note: This function imports TaskModelToResponse from task_mapper package
func DigestServiceResponseToDTO(digest *service.DigestResponse) *dto.DigestResponse {
	// Convert tasks from models to DTOs using TaskModelToResponse from task_mapper
	tasks := make([]dto.TaskResponse, len(digest.Tasks))
	for i, task := range digest.Tasks {
		// Import task_mapper functions - we need to call it directly
		// Since we can't import from same package, we'll inline the conversion
		tasks[i] = dto.TaskResponse{
			ID:                     task.ID,
			Title:                  task.Title,
			Description:            task.Description,
			UserID:                 task.UserID,
			MessengerRelatedUserID: task.MessengerRelatedUserID,
			ParentID:               task.ParentID,
			StartDate:              task.StartDate,
			FinishDate:             task.FinishDate,
			CronExpression:         task.CronExpression,
			RequiresConfirmation:   task.RequiresConfirmation,
			Status:                 task.Status,
			CreatedAt:              task.CreatedAt,
		}
	}

	return &dto.DigestResponse{
		UserID:                 digest.UserID,
		MessengerRelatedUserID: digest.MessengerRelatedUserID,
		ChatID:                 digest.ChatID,
		StartDateFrom:          digest.StartDateFrom,
		StartDateTo:            digest.StartDateTo,
		CompletedBacklogsCount: digest.CompletedBacklogsCount,
		Tasks:                  tasks,
		Timezone:               digest.Timezone,
	}
}
