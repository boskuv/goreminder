package models

import "time"

// TargetUpdateRequest represents a request to update a target item
// All fields are optional (pointers) to support partial updates
type TargetUpdateRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
