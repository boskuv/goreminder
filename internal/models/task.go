package models

import "time"

// Task represents the domain model for a task
type Task struct {
	ID          int64     `db:"id" json:"id"`
	Title       string    `db:"title" json:"title"`
	Description string    `db:"description" json:"description"`
	UserID      int64     `db:"user_id" json:"user_id"`
	DueDate     time.Time `db:"due_date" json:"due_date"`
	Status      string    `db:"status" json:"status"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}
