package mapper

import (
	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/models"
)

// CreateBacklogRequestToModel converts CreateBacklogRequest DTO to models.Backlog
func CreateBacklogRequestToModel(req *dto.CreateBacklogRequest) *models.Backlog {
	return &models.Backlog{
		Title:                  req.Title,
		Description:            req.Description,
		UserID:                 req.UserID,
		MessengerRelatedUserID: req.MessengerRelatedUserID,
	}
}

// UpdateBacklogRequestToModel converts UpdateBacklogRequest DTO to models.BacklogUpdateRequest
func UpdateBacklogRequestToModel(req *dto.UpdateBacklogRequest) *models.BacklogUpdateRequest {
	return &models.BacklogUpdateRequest{
		Title:       req.Title,
		Description: req.Description,
		CompletedAt: req.CompletedAt,
	}
}

// BacklogModelToResponse converts models.Backlog to BacklogResponse DTO
func BacklogModelToResponse(backlog *models.Backlog) *dto.BacklogResponse {
	return &dto.BacklogResponse{
		ID:                     backlog.ID,
		Title:                  backlog.Title,
		Description:            backlog.Description,
		UserID:                 backlog.UserID,
		MessengerRelatedUserID: backlog.MessengerRelatedUserID,
		CreatedAt:              backlog.CreatedAt,
		UpdatedAt:              backlog.UpdatedAt,
		CompletedAt:            backlog.CompletedAt,
	}
}

// BacklogsModelToResponse converts slice of models.Backlog to slice of BacklogResponse DTOs
func BacklogsModelToResponse(backlogs []*models.Backlog) []*dto.BacklogResponse {
	responses := make([]*dto.BacklogResponse, len(backlogs))
	for i, backlog := range backlogs {
		responses[i] = BacklogModelToResponse(backlog)
	}
	return responses
}
