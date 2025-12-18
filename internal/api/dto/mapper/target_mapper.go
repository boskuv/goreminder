package mapper

import (
	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/models"
)

// CreateTargetRequestToModel converts CreateTargetRequest DTO to models.Target
func CreateTargetRequestToModel(req *dto.CreateTargetRequest) *models.Target {
	return &models.Target{
		Title:                  req.Title,
		Description:            req.Description,
		UserID:                 req.UserID,
		MessengerRelatedUserID: req.MessengerRelatedUserID,
	}
}

// UpdateTargetRequestToModel converts UpdateTargetRequest DTO to models.TargetUpdateRequest
func UpdateTargetRequestToModel(req *dto.UpdateTargetRequest) *models.TargetUpdateRequest {
	return &models.TargetUpdateRequest{
		Title:       req.Title,
		Description: req.Description,
		CompletedAt: req.CompletedAt,
	}
}

// TargetModelToResponse converts models.Target to TargetResponse DTO
func TargetModelToResponse(target *models.Target) *dto.TargetResponse {
	return &dto.TargetResponse{
		ID:                     target.ID,
		Title:                  target.Title,
		Description:            target.Description,
		UserID:                 target.UserID,
		MessengerRelatedUserID: target.MessengerRelatedUserID,
		CreatedAt:              target.CreatedAt,
		UpdatedAt:              target.UpdatedAt,
		CompletedAt:            target.CompletedAt,
	}
}

// TargetsModelToResponse converts slice of models.Target to slice of TargetResponse DTOs
func TargetsModelToResponse(targets []*models.Target) []*dto.TargetResponse {
	responses := make([]*dto.TargetResponse, len(targets))
	for i, target := range targets {
		responses[i] = TargetModelToResponse(target)
	}
	return responses
}
