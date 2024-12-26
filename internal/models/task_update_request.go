package models

import "time"

// Task represents the domain model for a task
type TaskUpdateRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Status      *string    `json:"status,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty" example:"2024-12-01T00:00:00Z"`
}
