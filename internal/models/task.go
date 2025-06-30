package models

import "time"

// Task represents the domain model for a task
type Task struct {
	ID                     int64      `db:"id" json:"-"` // TODO: db?
	Title                  string     `db:"title" json:"title"`
	Description            string     `db:"description" json:"description"`
	UserID                 int64      `db:"user_id" json:"user_id"`
	MessengerRelatedUserID *int       `db:"messenger_related_user_id" json:"messenger_related_user_id,omitempty"`
	StartDate              time.Time  `db:"start_date" json:"start_date,omitempty"`
	FinishDate             *time.Time `db:"finish_date" json:"finish_date,omitempty"`
	CronExpression         *string    `db:"cron_expression" json:"cron_expression,omitempty"`
	Status                 string     `db:"status" json:"status"`
	CreatedAt              time.Time  `db:"created_at" json:"created_at"` // TODO: json:"-"`
	DeletedAt              time.Time  `db:"deleted_at" json:"-"`
}

// ScheduledTask represents the domain model for a task to schedule
type ScheduledTask struct {
	Action        string `json:"action" example:"add"`
	ChatID        string `json:"chat_id"`
	JobName       string `json:"job_name" example:"tasks.example_task"`
	MessengerName string `json:"messenger_name" example:"telegram"`
	QueueName     string `json:"queue_name" example:"celery"`
	TaskID        int64  `json:"task_id"`
}
