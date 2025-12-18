package dto

import "time"

// CreateTargetRequest represents the request DTO for creating a target item
type CreateTargetRequest struct {
	Title                  string `json:"title" binding:"required" example:"Learn Go programming"`
	Description            string `json:"description" example:"Master Go programming language"`
	UserID                 int64  `json:"user_id" binding:"required" example:"1"`
	MessengerRelatedUserID *int   `json:"messenger_related_user_id,omitempty" example:"123"`
}

// UpdateTargetRequest represents the request DTO for updating a target item
// All fields are optional (pointers) to support partial updates
type UpdateTargetRequest struct {
	Title       *string    `json:"title,omitempty" example:"Updated target title"`
	Description *string    `json:"description,omitempty" example:"Updated target description"`
	CompletedAt *time.Time `json:"completed_at,omitempty" example:"2024-01-15T10:00:00Z"`
}

// TargetResponse represents the response DTO for a target item
type TargetResponse struct {
	ID                     int64      `json:"id" example:"1"`
	Title                  string     `json:"title" example:"Learn Go programming"`
	Description            string     `json:"description" example:"Master Go programming language"`
	UserID                 int64      `json:"user_id" example:"1"`
	MessengerRelatedUserID *int       `json:"messenger_related_user_id,omitempty" example:"123"`
	CreatedAt              time.Time  `json:"created_at" example:"2024-01-10T08:00:00Z"`
	UpdatedAt              time.Time  `json:"updated_at" example:"2024-01-10T08:00:00Z"`
	CompletedAt            *time.Time `json:"completed_at,omitempty" example:"2024-01-15T10:00:00Z"`
}
