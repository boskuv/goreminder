package dto

// CreateMessengerRequest represents the request DTO for creating a messenger
type CreateMessengerRequest struct {
	Name string `json:"name" binding:"required"`
}

// MessengerResponse represents the response DTO for a messenger
type MessengerResponse struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"` // ISO 8601 format
}

// CreateMessengerRelatedUserRequest represents the request DTO for creating a messenger-related user
type CreateMessengerRelatedUserRequest struct {
	UserID          *int64 `json:"user_id" binding:"required"`
	MessengerID     *int64 `json:"messenger_id" binding:"required"`
	MessengerUserID string `json:"messenger_user_id" binding:"required"`
	ChatID          string `json:"chat_id" binding:"required"`
}

// MessengerRelatedUserResponse represents the response DTO for a messenger-related user
type MessengerRelatedUserResponse struct {
	ID              int64   `json:"id"`
	UserID          *int64  `json:"user_id"`
	MessengerID     *int64  `json:"messenger_id"`
	MessengerUserID string  `json:"messenger_user_id"`
	ChatID          string  `json:"chat_id"`
	CreatedAt       string  `json:"created_at"`           // ISO 8601 format
	UpdatedAt       *string `json:"updated_at,omitempty"` // ISO 8601 format
}
