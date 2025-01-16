package models

import "time"

// Messenger represents the domain model for a messenger
type Messenger struct {
	ID        int64     `db:"id" json:"-"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type MessengerRelatedUser struct {
	ID              int64      `db:"id" json:"-"`
	UserID          *int64     `db:"user_id" json:"user_id"`
	MessengerID     *int64     `db:"messenger_id" json:"messenger_id"`
	MessengerUserID string     `db:"messenger_user_id" json:"messenger_user_id"`
	ChatID          string     `db:"chat_id" json:"chat_id"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `db:"updated_at" json:"updated_at"`
}
