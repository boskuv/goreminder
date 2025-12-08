package dto

// CreateUserRequest represents the request DTO for creating a user
type CreateUserRequest struct {
	Name         string  `json:"name" binding:"required" example:"John Doe"`
	Email        string  `json:"email,omitempty" example:"john.doe@example.com"`
	PasswordHash string  `json:"password_hash,omitempty" example:"$2a$10$N9qo8uLOickgx2ZMRZoMye"`
	Timezone     *string `json:"timezone,omitempty" example:"UTC"`
	LanguageCode *string `json:"language_code,omitempty" example:"en"`
	Role         *string `json:"role,omitempty" example:"user" enums:"user,admin"`
}

// UpdateUserRequest represents the request DTO for updating a user
// All fields are optional (pointers) to support partial updates
type UpdateUserRequest struct {
	Name         *string `json:"name,omitempty" example:"Jane Doe"`
	Email        *string `json:"email,omitempty" binding:"omitempty,email" example:"jane.doe@example.com"`
	PasswordHash *string `json:"password_hash,omitempty" example:"$2a$10$N9qo8uLOickgx2ZMRZoMye"`
	Timezone     *string `json:"timezone,omitempty" example:"America/New_York"`
	LanguageCode *string `json:"language_code,omitempty" example:"ru"`
	Role         *string `json:"role,omitempty" example:"admin" enums:"user,admin"`
}

// UserResponse represents the response DTO for a user
// Note: PasswordHash and DeletedAt are excluded for security
type UserResponse struct {
	ID           int64   `json:"id" example:"1"`
	Name         string  `json:"name" example:"John Doe"`
	Email        string  `json:"email" example:"john.doe@example.com"`
	Timezone     *string `json:"timezone,omitempty" example:"UTC"`
	LanguageCode *string `json:"language_code,omitempty" example:"en"`
	Role         *string `json:"role,omitempty" example:"user" enums:"user,admin"`
	CreatedAt    string  `json:"created_at" example:"2024-01-10T08:00:00Z"` // ISO 8601 format
}
