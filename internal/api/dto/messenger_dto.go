package dto

// CreateMessengerRequest represents the request DTO for creating a messenger
type CreateMessengerRequest struct {
	Name string `json:"name" binding:"required" example:"telegram"`
}

// MessengerResponse represents the response DTO for a messenger
type MessengerResponse struct {
	Name      string `json:"name" example:"telegram"`
	CreatedAt string `json:"created_at" example:"2024-01-10T08:00:00Z"` // ISO 8601 format
}

// CreateMessengerRelatedUserRequest represents the request DTO for creating a messenger-related user
type CreateMessengerRelatedUserRequest struct {
	UserID          *int64 `json:"user_id" binding:"required" example:"1"`
	MessengerID     *int64 `json:"messenger_id" binding:"required" example:"1"`
	MessengerUserID string `json:"messenger_user_id" binding:"required" example:"123456789"`
	ChatID          string `json:"chat_id" binding:"required" example:"-1001234567890"`
}

// MessengerRelatedUserResponse represents the response DTO for a messenger-related user
type MessengerRelatedUserResponse struct {
	ID              int64   `json:"id" example:"1"`
	UserID          *int64  `json:"user_id" example:"1"`
	MessengerID     *int64  `json:"messenger_id" example:"1"`
	MessengerUserID string  `json:"messenger_user_id" example:"123456789"`
	ChatID          string  `json:"chat_id" example:"-1001234567890"`
	CreatedAt       string  `json:"created_at" example:"2024-01-10T08:00:00Z"`           // ISO 8601 format
	UpdatedAt       *string `json:"updated_at,omitempty" example:"2024-01-15T10:00:00Z"` // ISO 8601 format
}
