package models

import "time"

// Messenger represents the domain model for a messenger
type Messenger struct {
	ID        int64     `db:"id" json:"-"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type MessengerRelatedUser struct {
	ID          int       `json:"-"`
	UserID      *int      `json:"user_id"`
	MessengerID *int      `json:"messenger_id"`
	ChatID      string    `json:"chat_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
