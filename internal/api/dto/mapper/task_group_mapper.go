package mapper

import (
	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/models"
)

// CreateTaskGroupRequestToModel converts CreateTaskGroupRequest DTO to models.TaskGroup
func CreateTaskGroupRequestToModel(req *dto.CreateTaskGroupRequest) *models.TaskGroup {
	return &models.TaskGroup{
		Name:   req.Name,
		UserID: req.UserID,
	}
}

// UpdateTaskGroupRequestToModel converts UpdateTaskGroupRequest DTO to models.TaskGroupUpdateRequest
func UpdateTaskGroupRequestToModel(req *dto.UpdateTaskGroupRequest) *models.TaskGroupUpdateRequest {
	return &models.TaskGroupUpdateRequest{
		Name: req.Name,
	}
}

// TaskGroupModelToResponse converts models.TaskGroup to TaskGroupResponse DTO
func TaskGroupModelToResponse(group *models.TaskGroup) *dto.TaskGroupResponse {
	return &dto.TaskGroupResponse{
		ID:        group.ID,
		UserID:    group.UserID,
		Name:      group.Name,
		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	}
}

// TaskGroupsModelToResponse converts slice of models.TaskGroup to slice of TaskGroupResponse DTOs
func TaskGroupsModelToResponse(groups []*models.TaskGroup) []*dto.TaskGroupResponse {
	responses := make([]*dto.TaskGroupResponse, len(groups))
	for i, group := range groups {
		responses[i] = TaskGroupModelToResponse(group)
	}
	return responses
}
