package models

import "time"

// GoogleAccount stores a user's linked Google OAuth credentials (tokens encrypted at rest).
type GoogleAccount struct {
	ID              int64      `db:"id" json:"id"`
	UserID          int64      `db:"user_id" json:"user_id"`
	GoogleSub       string     `db:"google_sub" json:"google_sub"`
	Email           string     `db:"email" json:"email"`
	AccessTokenEnc  []byte     `db:"access_token_enc" json:"-"`
	RefreshTokenEnc []byte     `db:"refresh_token_enc" json:"-"`
	TokenExpiry     time.Time  `db:"token_expiry" json:"token_expiry"`
	Scopes          string     `db:"scopes" json:"scopes"`
	RevokedAt       *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}
