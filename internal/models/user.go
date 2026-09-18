package models

import "time"

type User struct {
	ID             int64      `db:"id" json:"id"`
	Name           string     `db:"name" json:"name"`
	Email          string     `db:"email" json:"email"`
	PasswordHash   string     `db:"password_hash" json:"-"` // Never expose in JSON
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	DeletedAt      time.Time  `db:"deleted_at" json:"-"` // Never expose in JSON
	Timezone       *string    `db:"timezone" json:"timezone,omitempty"`
	LanguageCode   *string    `db:"language_code" json:"language_code,omitempty"`
	Role           *string    `db:"role" json:"role,omitempty"`
	LastActivityAt *time.Time `db:"last_activity_at" json:"last_activity_at,omitempty"`
}

// UserActivity is a lightweight projection for recently active users.
type UserActivity struct {
	UserID         int64     `db:"id" json:"user_id"`
	Name           string    `db:"name" json:"name"`
	LastActivityAt time.Time `db:"last_activity_at" json:"last_activity_at"`
}
