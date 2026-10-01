package models

import "time"

// TaskSyncLinkOrigin indicates which side created the task↔event relationship.
type TaskSyncLinkOrigin string

const (
	TaskSyncLinkOriginImported TaskSyncLinkOrigin = "imported"
	TaskSyncLinkOriginExported TaskSyncLinkOrigin = "exported"
)

const TaskSyncProviderGoogleCalendar = "google_calendar"

// TaskSyncLink stores external calendar event linkage metadata for a task.
type TaskSyncLink struct {
	ID                int64              `db:"id" json:"id"`
	TaskID            int64              `db:"task_id" json:"task_id"`
	Provider          string             `db:"provider" json:"provider"`
	GoogleCalendarID  string             `db:"google_calendar_id" json:"google_calendar_id"`
	GoogleEventID     string             `db:"google_event_id" json:"google_event_id"`
	ETag              *string            `db:"etag" json:"etag,omitempty"`
	GoogleUpdatedAt   *time.Time         `db:"google_updated_at" json:"google_updated_at,omitempty"`
	Origin            TaskSyncLinkOrigin `db:"origin" json:"origin"`
	SyncEnabled       bool               `db:"sync_enabled" json:"sync_enabled"`
	// ExportOptIn is true when the user explicitly opted the task into export via
	// POST /tasks/{id}/calendar/export. Group-scoped exports leave this false so that
	// leaving the binding's group removes the Google event.
	ExportOptIn       bool               `db:"export_opt_in" json:"export_opt_in"`
	CalendarBindingID *int64             `db:"calendar_binding_id" json:"calendar_binding_id,omitempty"`
	DurationSeconds   *int               `db:"duration_seconds" json:"duration_seconds,omitempty"`
	LastSyncedAt      *time.Time         `db:"last_synced_at" json:"last_synced_at,omitempty"`
	LastError         *string            `db:"last_error" json:"last_error,omitempty"`
	CreatedAt         time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `db:"updated_at" json:"updated_at"`
}
