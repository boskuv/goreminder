package models

import "time"

// Task represents the domain model for a task
type TaskUpdateRequest struct {
	Title                *string    `json:"title,omitempty"`
	Description          *string    `json:"description,omitempty"`
	Status               *string    `json:"status,omitempty"`
	StartDate            *time.Time `json:"start_date,omitempty"`
	FinishDate           *time.Time `json:"finish_date,omitempty"`
	CronExpression       *string    `json:"cron_expression,omitempty"`
	RRule                *string    `json:"rrule,omitempty"`
	RequiresConfirmation *bool      `json:"requires_confirmation,omitempty"`
}
