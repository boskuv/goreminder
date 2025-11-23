package models

import "time"

type User struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"` // Never expose in JSON
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	DeletedAt    time.Time `db:"deleted_at" json:"-"` // Never expose in JSON
	Timezone     *string   `db:"timezone" json:"timezone,omitempty"`
	LanguageCode *string   `db:"language_code" json:"language_code,omitempty"`
	Role         *string   `db:"role" json:"role,omitempty"`
}
