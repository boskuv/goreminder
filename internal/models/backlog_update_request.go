package models

import "time"

// BacklogUpdateRequest represents a request to update a backlog item
// All fields are optional (pointers) to support partial updates
type BacklogUpdateRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
