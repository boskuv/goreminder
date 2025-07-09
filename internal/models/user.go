package models

import "time"

type User struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
	DeletedAt    time.Time `db:"deleted_at" json:"-"`
	Timezone     *string   `db:"timezone" json:"timezone,omitempty"`
	LanguageCode *string   `db:"language_code" json:"language_code,omitempty"`
	Role         *string   `db:"role" json:"role,omitempty"`
}
