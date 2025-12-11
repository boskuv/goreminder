package dto

import "time"

// CreateBacklogRequest represents the request DTO for creating a backlog item
type CreateBacklogRequest struct {
	Title                  string `json:"title" binding:"required" example:"Implement new feature"`
	Description            string `json:"description" example:"Add user authentication"`
	UserID                 int64  `json:"user_id" binding:"required" example:"1"`
	MessengerRelatedUserID *int   `json:"messenger_related_user_id,omitempty" example:"123"`
}

// CreateBacklogsBatchRequest represents the request DTO for creating multiple backlog items
// Items are separated by separator (default: newline)
type CreateBacklogsBatchRequest struct {
	Items                  string `json:"items" binding:"required" example:"Item 1\nItem 2\nItem 3"`
	Separator              string `json:"separator,omitempty" example:"\n"`
	UserID                 int64  `json:"user_id" binding:"required" example:"1"`
	MessengerRelatedUserID *int   `json:"messenger_related_user_id,omitempty" example:"123"`
}

// UpdateBacklogRequest represents the request DTO for updating a backlog item
// All fields are optional (pointers) to support partial updates
type UpdateBacklogRequest struct {
	Title       *string    `json:"title,omitempty" example:"Updated backlog title"`
	Description *string    `json:"description,omitempty" example:"Updated backlog description"`
	CompletedAt *time.Time `json:"completed_at,omitempty" example:"2024-01-15T10:00:00Z"`
}

// BacklogResponse represents the response DTO for a backlog item
type BacklogResponse struct {
	ID                     int64      `json:"id" example:"1"`
	Title                  string     `json:"title" example:"Implement new feature"`
	Description            string     `json:"description" example:"Add user authentication"`
	UserID                 int64      `json:"user_id" example:"1"`
	MessengerRelatedUserID *int       `json:"messenger_related_user_id,omitempty" example:"123"`
	CreatedAt              time.Time  `json:"created_at" example:"2024-01-10T08:00:00Z"`
	UpdatedAt              time.Time  `json:"updated_at" example:"2024-01-10T08:00:00Z"`
	CompletedAt            *time.Time `json:"completed_at,omitempty" example:"2024-01-15T10:00:00Z"`
}
