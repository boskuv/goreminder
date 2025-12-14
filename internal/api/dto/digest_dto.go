package dto

import "time"

// CreateDigestSettingsRequest represents the request DTO for creating digest settings
type CreateDigestSettingsRequest struct {
	UserID                 int64  `json:"user_id" binding:"required" example:"1"`
	MessengerRelatedUserID *int   `json:"messenger_related_user_id,omitempty" example:"123"`
	Enabled                bool   `json:"enabled" example:"true"`
	WeekdayTime            string `json:"weekday_time" binding:"required" example:"07:00"` // Format: HH:MM
	WeekendTime            string `json:"weekend_time" binding:"required" example:"10:00"` // Format: HH:MM
}

// UpdateDigestSettingsRequest represents the request DTO for updating digest settings
// All fields are optional (pointers) to support partial updates
type UpdateDigestSettingsRequest struct {
	Enabled                *bool   `json:"enabled,omitempty" example:"true"`
	WeekdayTime            *string `json:"weekday_time,omitempty" example:"08:00"` // Format: HH:MM
	WeekendTime            *string `json:"weekend_time,omitempty" example:"11:00"` // Format: HH:MM
	MessengerRelatedUserID *int    `json:"messenger_related_user_id,omitempty" example:"123"`
}

// DigestSettingsResponse represents the response DTO for digest settings
type DigestSettingsResponse struct {
	ID                     int64  `json:"id" example:"1"`
	UserID                 int64  `json:"user_id" example:"1"`
	MessengerRelatedUserID *int   `json:"messenger_related_user_id,omitempty" example:"123"`
	Enabled                bool   `json:"enabled" example:"true"`
	WeekdayTime            string `json:"weekday_time" example:"07:00"`
	WeekendTime            string `json:"weekend_time" example:"10:00"`
	CreatedAt              string `json:"created_at" example:"2024-01-10T08:00:00Z"` // ISO 8601 format
	UpdatedAt              string `json:"updated_at" example:"2024-01-10T08:00:00Z"` // ISO 8601 format
}

// DigestResponse represents the response DTO for a digest
type DigestResponse struct {
	UserID                 int64          `json:"user_id" example:"1"`
	MessengerRelatedUserID *int           `json:"messenger_related_user_id,omitempty" example:"123"`
	ChatID                 *string        `json:"chat_id,omitempty" example:"123"`
	StartDateFrom          time.Time      `json:"start_date_from" example:"2024-01-01T00:00:00Z"`
	StartDateTo            time.Time      `json:"start_date_to" example:"2024-01-07T23:59:59Z"`
	CompletedBacklogsCount int            `json:"completed_backlogs_count" example:"5"`
	Tasks                  []TaskResponse `json:"tasks"`
	Timezone               string         `json:"timezone" example:"UTC"`
}

// PaginatedDigestSettingsResponse represents a paginated response for digest settings
type PaginatedDigestSettingsResponse struct {
	Data       []DigestSettingsResponse `json:"data"`
	Pagination PaginationResponse       `json:"pagination"`
}
