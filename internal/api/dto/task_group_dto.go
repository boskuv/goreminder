package dto

import "time"

// CreateTaskGroupRequest represents the request DTO for creating a task group
type CreateTaskGroupRequest struct {
	Name   string `json:"name" binding:"required" example:"Work"`
	UserID int64  `json:"user_id" binding:"required" example:"1"`
}

// UpdateTaskGroupRequest represents the request DTO for updating a task group
// All fields are optional (pointers) to support partial updates
type UpdateTaskGroupRequest struct {
	Name *string `json:"name,omitempty" example:"Personal"`
}

// TaskGroupResponse represents the response DTO for a task group
type TaskGroupResponse struct {
	ID        int64     `json:"id" example:"1"`
	UserID    int64     `json:"user_id" example:"1"`
	Name      string    `json:"name" example:"Work"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-10T08:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2024-01-10T08:00:00Z"`
}
