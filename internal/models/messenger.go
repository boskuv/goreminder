package models

import "time"

// Messenger represents the domain model for a messenger
type Messenger struct {
	ID        int64     `db:"id" json:"-"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
