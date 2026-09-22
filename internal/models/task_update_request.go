package models

import "time"

// TaskUpdateRequest represents a request to update a task.
// All fields are optional (pointers) to support partial updates.
type TaskUpdateRequest struct {
	Title                *string    `json:"title,omitempty"`
	Description          *string    `json:"description,omitempty"`
	Status               *string    `json:"status,omitempty"`
	StartDate            *time.Time `json:"start_date,omitempty"`
	FinishDate           *time.Time `json:"finish_date,omitempty"`
	CronExpression       *string    `json:"cron_expression,omitempty"`
	RRule                *string    `json:"rrule,omitempty"`
	RequiresConfirmation *bool      `json:"requires_confirmation,omitempty"`
	Muted                *bool      `json:"muted,omitempty"`
	// PreRemindBeforeSeconds: omit = no change; 0 = clear; >0 = set offset in seconds.
	PreRemindBeforeSeconds *int64 `json:"pre_remind_before_seconds,omitempty"`
	// GroupID: omit = no change; 0 = clear group; >0 = assign to group.
	GroupID *int64 `json:"group_id,omitempty"`
}
