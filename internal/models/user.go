package models

import "time"

type User struct {
	ID           int64     `db:"id" json:"-"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
	DeletedAt    time.Time `db:"deleted_at" json:"-"`
	Timezone     *string   `db:"timezone" json:"timezone,omitempty"`
}
