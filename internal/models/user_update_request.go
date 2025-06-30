package models

type UserUpdateRequest struct {
	Name         *string `json:"name,omitempty"`
	Email        *string `json:"email,omitempty"`
	PasswordHash *string `json:"password_hash,omitempty"`
	Timezone     *string `json:"timezone,omitempty"`
	LanguageCode *string `json:"language_code,omitempty"`
	Role         *string `json:"role,omitempty"`
}
