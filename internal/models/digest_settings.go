package models

import "time"

// DigestSettings represents the domain model for digest settings
type DigestSettings struct {
	ID                     int64     `db:"id" json:"id"`
	UserID                 int64     `db:"user_id" json:"user_id"`
	MessengerRelatedUserID *int      `db:"messenger_related_user_id" json:"messenger_related_user_id,omitempty"`
	Enabled                bool      `db:"enabled" json:"enabled"`
	WeekdayTime            string    `db:"weekday_time" json:"weekday_time"` // Format: "HH:MM" (e.g., "07:00") UTC
	WeekendTime            string    `db:"weekend_time" json:"weekend_time"` // Format: "HH:MM" (e.g., "10:00") UTC
	CreatedAt              time.Time `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time `db:"updated_at" json:"updated_at"`
}

// DigestSettingsUpdateRequest represents a request to update digest settings
type DigestSettingsUpdateRequest struct {
	Enabled                *bool   `json:"enabled,omitempty"`
	WeekdayTime            *string `json:"weekday_time,omitempty"` // Format: "HH:MM" UTC
	WeekendTime            *string `json:"weekend_time,omitempty"` // Format: "HH:MM" UTC
	MessengerRelatedUserID *int    `json:"messenger_related_user_id,omitempty"`
}
