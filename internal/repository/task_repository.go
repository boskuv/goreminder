// internal/repository/task_repository.go
package repository

import (
	"database/sql"
	"fmt"
)

type Task struct {
	ID          int64
	Title       string
	Description string
	UserID      int64
	DueDate     string
	Status      string
}

type TaskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a new instance of TaskRepository
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// CreateTask inserts a new task into the database
func (r *TaskRepository) CreateTask(task *Task) (int64, error) {
	query := `
		INSERT INTO tasks (title, description, user_id, due_date, status)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`

	var id int64
	err := r.db.QueryRow(query, task.Title, task.Description, task.UserID, task.DueDate, task.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create task: %w", err)
	}

	return id, nil
}

// GetTaskByID fetches a task by its ID
func (r *TaskRepository) GetTaskByID(id int64) (*Task, error) {
	query := `SELECT id, title, description, user_id, due_date, status FROM tasks WHERE id = $1`

	task := &Task{}
	err := r.db.QueryRow(query, id).Scan(&task.ID, &task.Title, &task.Description, &task.UserID, &task.DueDate, &task.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task by id: %w", err)
	}

	return task, nil
}

// UpdateTask updates a task in the database
func (r *TaskRepository) UpdateTask(task *Task) error {
	query := `
		UPDATE tasks
		SET title = $1, description = $2, user_id = $3, due_date = $4, status = $5
		WHERE id = $6
	`

	_, err := r.db.Exec(query, task.Title, task.Description, task.UserID, task.DueDate, task.Status, task.ID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// DeleteTask deletes a task by its ID
func (r *TaskRepository) DeleteTask(id int64) error {
	query := `DELETE FROM tasks WHERE id = $1`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}
