package models

import (
	"fmt"
	"time"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	// TaskStatusPending - task created, awaiting execution
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusScheduled - task scheduled (in queue/calendar)
	TaskStatusScheduled TaskStatus = "scheduled"
	// TaskStatusDone - task completed
	TaskStatusDone TaskStatus = "done"
	// TaskStatusRescheduled - task rescheduled (for one-time tasks with autoreschedule)
	TaskStatusRescheduled TaskStatus = "rescheduled"
	// TaskStatusPostponed - task postponed
	TaskStatusPostponed TaskStatus = "postponed"
	// TaskStatusDeleted - task deleted
	TaskStatusDeleted TaskStatus = "deleted"
)

// ValidTaskStatuses returns a slice of all valid task statuses
func ValidTaskStatuses() []TaskStatus {
	return []TaskStatus{
		TaskStatusPending,
		TaskStatusScheduled,
		TaskStatusDone,
		TaskStatusRescheduled,
		TaskStatusPostponed,
		TaskStatusDeleted,
	}
}

// IsValid checks if the status is valid
func (s TaskStatus) IsValid() bool {
	for _, validStatus := range ValidTaskStatuses() {
		if s == validStatus {
			return true
		}
	}
	return false
}

// String returns the string representation of the status
func (s TaskStatus) String() string {
	return string(s)
}

// ValidateTaskStatus validates if a status string is valid
func ValidateTaskStatus(status string) error {
	taskStatus := TaskStatus(status)
	if !taskStatus.IsValid() {
		return fmt.Errorf("invalid task status: %s. Valid statuses are: %v", status, ValidTaskStatuses())
	}
	return nil
}

// Task represents the domain model for a task
type Task struct {
	ID                     int64      `db:"id" json:"id"`
	Title                  string     `db:"title" json:"title"`
	Description            string     `db:"description" json:"description"`
	UserID                 int64      `db:"user_id" json:"user_id"`
	MessengerRelatedUserID *int       `db:"messenger_related_user_id" json:"messenger_related_user_id,omitempty"`
	ParentID               *int64     `db:"parent_id" json:"parent_id,omitempty"`
	StartDate              time.Time  `db:"start_date" json:"start_date,omitempty"`
	FinishDate             *time.Time `db:"finish_date" json:"finish_date,omitempty"`
	CronExpression         *string    `db:"cron_expression" json:"cron_expression,omitempty"`
	RRule                  *string    `db:"rrule" json:"rrule,omitempty"`
	RequiresConfirmation   bool       `db:"requires_confirmation" json:"requires_confirmation,omitempty"`
	Muted                  bool       `db:"muted" json:"muted,omitempty"`
	Status                 string     `db:"status" json:"status"`
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
	DeletedAt              time.Time  `db:"deleted_at" json:"-"`
}

// ScheduledTask represents the domain model for a task to enqueue
type ScheduledTask struct {
	Action string `json:"action" example:"schedule"`
	// MessengerName string `json:"messenger_name" example:"telegram"` TODO
	QueueName string `json:"queue_name" example:"celery"`
	TaskID    int64  `json:"task_id"`
}

const (
	ScheduledTaskActionSchedule = "schedule"
	ScheduledTaskActionDelete   = "delete"
)
