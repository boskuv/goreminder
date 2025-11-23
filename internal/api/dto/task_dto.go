package dto

import "time"

// CreateTaskRequest represents the request DTO for creating a task
type CreateTaskRequest struct {
	Title                  string     `json:"title" binding:"required"`
	Description            string     `json:"description"`
	UserID                 int64      `json:"user_id" binding:"required"`
	MessengerRelatedUserID *int       `json:"messenger_related_user_id,omitempty"`
	StartDate              time.Time  `json:"start_date"`
	FinishDate             *time.Time `json:"finish_date,omitempty"`
	CronExpression         *string    `json:"cron_expression,omitempty"`
	Status                 string     `json:"status,omitempty"`
}

// UpdateTaskRequest represents the request DTO for updating a task
// All fields are optional (pointers) to support partial updates
type UpdateTaskRequest struct {
	Title          *string    `json:"title,omitempty"`
	Description    *string    `json:"description,omitempty"`
	Status         *string    `json:"status,omitempty"`
	StartDate      *time.Time `json:"start_date,omitempty"`
	FinishDate     *time.Time `json:"finish_date,omitempty"`
	CronExpression *string    `json:"cron_expression,omitempty"`
}

// TaskResponse represents the response DTO for a task
type TaskResponse struct {
	ID                     int64      `json:"id"`
	Title                  string     `json:"title"`
	Description            string     `json:"description"`
	UserID                 int64      `json:"user_id"`
	MessengerRelatedUserID *int       `json:"messenger_related_user_id,omitempty"`
	StartDate              time.Time  `json:"start_date"`
	FinishDate             *time.Time `json:"finish_date,omitempty"`
	CronExpression         *string    `json:"cron_expression,omitempty"`
	Status                 string     `json:"status"`
	CreatedAt              time.Time  `json:"created_at"`
}

// QueueTaskRequest represents the request DTO for queuing a task
type QueueTaskRequest struct {
	Action    string `json:"action" binding:"required" example:"schedule"`
	QueueName string `json:"queue_name" binding:"required" example:"celery"`
	TaskID    int64  `json:"task_id" binding:"required"`
}
