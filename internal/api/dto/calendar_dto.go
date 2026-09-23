package dto

import "time"

// OAuthStartResponse is returned when starting Google OAuth.
type OAuthStartResponse struct {
	URL string `json:"url" example:"https://accounts.google.com/o/oauth2/auth?..."`
}

// GoogleCalendarListItem is a calendar from the connected Google account.
type GoogleCalendarListItem struct {
	ID      string `json:"id" example:"primary"`
	Summary string `json:"summary" example:"Work"`
	Primary bool   `json:"primary" example:"true"`
}

// CreateCalendarBindingRequest creates a sync binding for a Google calendar.
type CreateCalendarBindingRequest struct {
	GoogleCalendarID         string  `json:"google_calendar_id" binding:"required" example:"primary"`
	CalendarSummary          *string `json:"calendar_summary,omitempty" example:"Work"`
	Direction                string  `json:"direction,omitempty" example:"import" enums:"import,export,both"`
	GroupID                  *int64  `json:"group_id,omitempty" example:"1"`
	MessengerRelatedUserID   *int    `json:"messenger_related_user_id,omitempty" example:"1"`
	DeletePolicy             string  `json:"delete_policy,omitempty" example:"soft_delete_imported" enums:"soft_delete_imported,mute_imported,keep"`
}

// CalendarBindingResponse is the API representation of a calendar binding.
type CalendarBindingResponse struct {
	ID                       int64      `json:"id" example:"1"`
	UserID                   int64      `json:"user_id" example:"1"`
	GoogleAccountID          int64      `json:"google_account_id" example:"1"`
	GoogleCalendarID         string     `json:"google_calendar_id" example:"primary"`
	CalendarSummary          *string    `json:"calendar_summary,omitempty" example:"Work"`
	Direction                string     `json:"direction" example:"import"`
	GroupID                  *int64     `json:"group_id,omitempty" example:"1"`
	MessengerRelatedUserID   *int       `json:"messenger_related_user_id,omitempty" example:"1"`
	LastSyncedAt             *time.Time `json:"last_synced_at,omitempty"`
	LastError                *string    `json:"last_error,omitempty"`
	Status                   string     `json:"status" example:"active"`
	DeletePolicy             string     `json:"delete_policy" example:"soft_delete_imported"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// GoogleAccountResponse is a safe view of a connected Google account (no tokens).
type GoogleAccountResponse struct {
	ID        int64      `json:"id" example:"1"`
	UserID    int64      `json:"user_id" example:"1"`
	GoogleSub string     `json:"google_sub" example:"123456789"`
	Email     string     `json:"email" example:"user@gmail.com"`
	Scopes    string     `json:"scopes"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// EnableTaskExportRequest opts a single task into export for a binding.
type EnableTaskExportRequest struct {
	CalendarBindingID int64 `json:"calendar_binding_id" binding:"required" example:"1"`
}

// TaskExternalResponse marks a task as linked to an external calendar event.
type TaskExternalResponse struct {
	Provider         string     `json:"provider" example:"google_calendar"`
	CalendarID       string     `json:"calendar_id" example:"primary"`
	EventID          string     `json:"event_id" example:"abc123"`
	Origin           string     `json:"origin" example:"imported" enums:"imported,exported"`
	SyncEnabled      bool       `json:"sync_enabled" example:"true"`
	LastSyncedAt     *time.Time `json:"last_synced_at,omitempty"`
	LastError        *string    `json:"last_error,omitempty"`
	CalendarBindingID *int64    `json:"calendar_binding_id,omitempty"`
}
