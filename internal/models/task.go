package models

import "time"

// Task represents the domain model for a task
type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	UserID      int64     `json:"user_id" binding:"required"`
	DueDate     time.Time `json:"due_date" binding:"required"`
	Status      string    `json:"status" binding:"required,oneof=pending completed"`
	CreatedAt   time.Time `json:"created_at"`
}
