package dto

import "time"

// CreateTaskRequest represents the request DTO for creating a task
type CreateTaskRequest struct {
	Title                  string     `json:"title" binding:"required" example:"Complete project documentation"`
	Description            string     `json:"description" example:"Write comprehensive documentation for the API"`
	UserID                 int64      `json:"user_id" binding:"required" example:"1"`
	MessengerRelatedUserID *int       `json:"messenger_related_user_id,omitempty" example:"123"`
	StartDate              time.Time  `json:"start_date" example:"2024-01-15T10:00:00Z"`
	FinishDate             *time.Time `json:"finish_date,omitempty" example:"2024-01-20T18:00:00Z"`
	CronExpression         *string    `json:"cron_expression,omitempty" binding:"omitempty,cron" example:"0 9 * * *"`
	Status                 string     `json:"status,omitempty" binding:"omitempty,task_status" example:"pending" enums:"pending,scheduled,done,rescheduled,postponed,deleted"`
}

// UpdateTaskRequest represents the request DTO for updating a task
// All fields are optional (pointers) to support partial updates
type UpdateTaskRequest struct {
	Title          *string    `json:"title,omitempty" example:"Updated task title"`
	Description    *string    `json:"description,omitempty" example:"Updated task description"`
	Status         *string    `json:"status,omitempty" binding:"omitempty,task_status" example:"done" enums:"pending,scheduled,done,rescheduled,postponed,deleted"`
	StartDate      *time.Time `json:"start_date,omitempty" example:"2024-01-15T10:00:00Z"`
	FinishDate     *time.Time `json:"finish_date,omitempty" example:"2024-01-20T18:00:00Z"`
	CronExpression *string    `json:"cron_expression,omitempty" binding:"omitempty,cron" example:"0 9 * * *"`
}

// TaskResponse represents the response DTO for a task
type TaskResponse struct {
	ID                     int64      `json:"id" example:"1"`
	Title                  string     `json:"title" example:"Complete project documentation"`
	Description            string     `json:"description" example:"Write comprehensive documentation for the API"`
	UserID                 int64      `json:"user_id" example:"1"`
	MessengerRelatedUserID *int       `json:"messenger_related_user_id,omitempty" example:"123"`
	StartDate              time.Time  `json:"start_date" example:"2024-01-15T10:00:00Z"`
	FinishDate             *time.Time `json:"finish_date,omitempty" example:"2024-01-20T18:00:00Z"`
	CronExpression         *string    `json:"cron_expression,omitempty" example:"0 9 * * *"`
	Status                 string     `json:"status" example:"pending" enums:"pending,scheduled,done,rescheduled,postponed,deleted"`
	CreatedAt              time.Time  `json:"created_at" example:"2024-01-10T08:00:00Z"`
}

// QueueTaskRequest represents the request DTO for queuing a task
type QueueTaskRequest struct {
	Action    string `json:"action" binding:"required" example:"schedule" enums:"schedule,delete"`
	QueueName string `json:"queue_name" binding:"required" example:"celery"`
	TaskID    int64  `json:"task_id" binding:"required" example:"1"`
}
