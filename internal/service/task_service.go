package service

import (
	"fmt"
	"time"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
)

// TaskService defines methods for task-related business logic
type TaskService struct {
	TaskRepo repository.TaskRepository
	UserRepo repository.UserRepository
}

// NewTaskService creates a new TaskService
func NewTaskService(taskRepo repository.TaskRepository, userRepo repository.UserRepository) *TaskService {
	return &TaskService{
		TaskRepo: taskRepo,
		UserRepo: userRepo,
	}
}

// CreateTask creates a new task
func (s *TaskService) CreateTask(task *models.Task) (int64, error) {
	// Check if the user exists
	user, err := s.UserRepo.GetUserByID(task.UserID)
	if err != nil {
		return 0, fmt.Errorf("failed to verify user: %w", err)
	}
	if user == nil {
		return 0, fmt.Errorf("user with ID %d does not exist", task.UserID)
	}

	// Set default values
	task.Status = "pending"
	task.CreatedAt = time.Now()

	// Save the task
	taskID, err := s.TaskRepo.CreateTask(task)
	if err != nil {
		return 0, fmt.Errorf("failed to create task: %w", err)
	}

	return taskID, nil
}

// GetTask retrieves a task by its ID
func (s *TaskService) GetTask(taskID int64) (*models.Task, error) {
	task, err := s.TaskRepo.GetTaskByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task with ID %d does not exist", taskID)
	}
	return task, nil
}
