package models

import "time"

// CalendarBindingDirection controls sync direction for a calendar binding.
type CalendarBindingDirection string

const (
	CalendarBindingDirectionImport CalendarBindingDirection = "import"
	CalendarBindingDirectionExport CalendarBindingDirection = "export"
	CalendarBindingDirectionBoth   CalendarBindingDirection = "both"
)

// CalendarBindingStatus is the operational status of a calendar binding.
type CalendarBindingStatus string

const (
	CalendarBindingStatusActive       CalendarBindingStatus = "active"
	CalendarBindingStatusError        CalendarBindingStatus = "error"
	CalendarBindingStatusDisconnected CalendarBindingStatus = "disconnected"
)

// CalendarDeletePolicy controls what happens to imported tasks when a binding is removed.
type CalendarDeletePolicy string

const (
	CalendarDeletePolicySoftDeleteImported CalendarDeletePolicy = "soft_delete_imported"
	CalendarDeletePolicyMuteImported       CalendarDeletePolicy = "mute_imported"
	CalendarDeletePolicyKeep               CalendarDeletePolicy = "keep"
)

// CalendarBinding links a Google calendar to a user (and optionally a task group).
type CalendarBinding struct {
	ID               int64                    `db:"id" json:"id"`
	UserID           int64                    `db:"user_id" json:"user_id"`
	GoogleAccountID  int64                    `db:"google_account_id" json:"google_account_id"`
	GoogleCalendarID string                   `db:"google_calendar_id" json:"google_calendar_id"`
	CalendarSummary  *string                  `db:"calendar_summary" json:"calendar_summary,omitempty"`
	Direction        CalendarBindingDirection `db:"direction" json:"direction"`
	GroupID                  *int64                   `db:"group_id" json:"group_id,omitempty"`
	MessengerRelatedUserID   *int                     `db:"messenger_related_user_id" json:"messenger_related_user_id,omitempty"`
	SyncToken                *string                  `db:"sync_token" json:"-"`
	LastSyncedAt             *time.Time               `db:"last_synced_at" json:"last_synced_at,omitempty"`
	LastError                *string                  `db:"last_error" json:"last_error,omitempty"`
	Status                   CalendarBindingStatus    `db:"status" json:"status"`
	DeletePolicy             CalendarDeletePolicy     `db:"delete_policy" json:"delete_policy"`
	SyncAttempts             int                      `db:"sync_attempts" json:"sync_attempts"`
	NextRetryAt              *time.Time               `db:"next_retry_at" json:"next_retry_at,omitempty"`
	CreatedAt                time.Time                `db:"created_at" json:"created_at"`
	UpdatedAt                time.Time                `db:"updated_at" json:"updated_at"`
	DeletedAt                time.Time                `db:"deleted_at" json:"-"`
}
