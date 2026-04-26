package mapper

import (
	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/models"
)

// CreateTaskRequestToModel converts CreateTaskRequest DTO to models.Task
func CreateTaskRequestToModel(req *dto.CreateTaskRequest) *models.Task {
	task := &models.Task{
		Title:                  req.Title,
		Description:            req.Description,
		UserID:                 req.UserID,
		MessengerRelatedUserID: req.MessengerRelatedUserID,
		StartDate:              req.StartDate,
		FinishDate:             req.FinishDate,
		CronExpression:         req.CronExpression,
		RRule:                  req.RRule,
		RequiresConfirmation:   req.RequiresConfirmation,
		Status:                 req.Status,
	}
	if task.Status == "" {
		task.Status = string(models.TaskStatusPending)
	}
	return task
}

// UpdateTaskRequestToModel converts UpdateTaskRequest DTO to models.TaskUpdateRequest
func UpdateTaskRequestToModel(req *dto.UpdateTaskRequest) *models.TaskUpdateRequest {
	return &models.TaskUpdateRequest{
		Title:                req.Title,
		Description:          req.Description,
		Status:               req.Status,
		StartDate:            req.StartDate,
		FinishDate:           req.FinishDate,
		CronExpression:       req.CronExpression,
		RRule:                req.RRule,
		RequiresConfirmation: req.RequiresConfirmation,
	}
}

// TaskModelToResponse converts models.Task to TaskResponse DTO
func TaskModelToResponse(task *models.Task) *dto.TaskResponse {
	return &dto.TaskResponse{
		ID:                     task.ID,
		Title:                  task.Title,
		Description:            task.Description,
		UserID:                 task.UserID,
		MessengerRelatedUserID: task.MessengerRelatedUserID,
		ParentID:               task.ParentID,
		StartDate:              task.StartDate,
		FinishDate:             task.FinishDate,
		CronExpression:         task.CronExpression,
		RRule:                  task.RRule,
		RequiresConfirmation:   task.RequiresConfirmation,
		Status:                 task.Status,
		CreatedAt:              task.CreatedAt,
	}
}

// TaskModelToMarkedDoneResponse converts models.Task to TaskMarkedDoneResponse DTO (no status field).
func TaskModelToMarkedDoneResponse(task *models.Task) *dto.TaskMarkedDoneResponse {
	return &dto.TaskMarkedDoneResponse{
		ID:                     task.ID,
		Title:                  task.Title,
		Description:            task.Description,
		UserID:                 task.UserID,
		MessengerRelatedUserID: task.MessengerRelatedUserID,
		ParentID:               task.ParentID,
		StartDate:              task.StartDate,
		FinishDate:             task.FinishDate,
		CronExpression:         task.CronExpression,
		RRule:                  task.RRule,
		RequiresConfirmation:   task.RequiresConfirmation,
		CreatedAt:              task.CreatedAt,
	}
}

// TasksModelToResponse converts slice of models.Task to slice of TaskResponse DTOs
func TasksModelToResponse(tasks []*models.Task) []*dto.TaskResponse {
	responses := make([]*dto.TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = TaskModelToResponse(task)
	}
	return responses
}

// QueueTaskRequestToModel converts QueueTaskRequest DTO to models.ScheduledTask
func QueueTaskRequestToModel(req *dto.QueueTaskRequest) *models.ScheduledTask {
	return &models.ScheduledTask{
		Action:    req.Action,
		QueueName: req.QueueName,
		TaskID:    req.TaskID,
	}
}
