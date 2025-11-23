package dto

// CreateUserRequest represents the request DTO for creating a user
type CreateUserRequest struct {
	Name         string  `json:"name" binding:"required"`
	Email        string  `json:"email" binding:"required"`
	PasswordHash string  `json:"password_hash" binding:"required"`
	Timezone     *string `json:"timezone,omitempty"`
	LanguageCode *string `json:"language_code,omitempty"`
	Role         *string `json:"role,omitempty"`
}

// UpdateUserRequest represents the request DTO for updating a user
// All fields are optional (pointers) to support partial updates
type UpdateUserRequest struct {
	Name         *string `json:"name,omitempty"`
	Email        *string `json:"email,omitempty"`
	PasswordHash *string `json:"password_hash,omitempty"`
	Timezone     *string `json:"timezone,omitempty"`
	LanguageCode *string `json:"language_code,omitempty"`
	Role         *string `json:"role,omitempty"`
}

// UserResponse represents the response DTO for a user
// Note: PasswordHash and DeletedAt are excluded for security
type UserResponse struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Timezone     *string `json:"timezone,omitempty"`
	LanguageCode *string `json:"language_code,omitempty"`
	Role         *string `json:"role,omitempty"`
	CreatedAt    string  `json:"created_at"` // ISO 8601 format
}
