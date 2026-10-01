package models

import (
	"time"
)

// TaskHistoryAction represents the type of action performed on a task
type TaskHistoryAction string

const (
	TaskHistoryActionCreated            TaskHistoryAction = "created"
	TaskHistoryActionUpdated            TaskHistoryAction = "updated"
	TaskHistoryActionDeleted            TaskHistoryAction = "deleted"
	TaskHistoryActionStatusChanged      TaskHistoryAction = "status_changed"
	TaskHistoryActionAttachmentAdded    TaskHistoryAction = "attachment_added"
	TaskHistoryActionAttachmentRemoved  TaskHistoryAction = "attachment_removed"
)

// Task history new_value/old_value keys used by calendar import to mark provenance without a new action enum.
const (
	TaskHistorySourceKey            = "source"
	TaskHistorySourceGoogleCalendar = "google_calendar_import"
)

// TaskHistory represents the history entry for a task
type TaskHistory struct {
	ID        int64                  `db:"id" json:"id"`
	TaskID    int64                  `db:"task_id" json:"task_id"`
	UserID    int64                  `db:"user_id" json:"user_id"`
	Action    string                 `db:"action" json:"action"`
	OldValue  map[string]interface{} `db:"old_value" json:"old_value,omitempty"`
	NewValue  map[string]interface{} `db:"new_value" json:"new_value,omitempty"`
	CreatedAt time.Time              `db:"created_at" json:"created_at"`
}
