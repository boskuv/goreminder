package models

import "time"

// Backlog represents the domain model for a backlog item
type Backlog struct {
	ID                     int64      `db:"id" json:"id"`
	Title                  string     `db:"title" json:"title"`
	Description            string     `db:"description" json:"description"`
	UserID                 int64      `db:"user_id" json:"user_id"`
	MessengerRelatedUserID *int       `db:"messenger_related_user_id" json:"messenger_related_user_id,omitempty"`
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time  `db:"updated_at" json:"updated_at"`
	CompletedAt            *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	DeletedAt              time.Time  `db:"deleted_at" json:"-"`
}
