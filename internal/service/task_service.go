package service

import (
	"time"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/pkg/errors"
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
		return 0, err
	}

	if user == nil {
		return 0, errors.WithStack(errors.Errorf("user with ID %d does not exist", task.UserID))
	}

	// Set default values
	task.Status = "pending"
	task.CreatedAt = time.Now() // TODO: time format

	// Save the task
	taskID, err := s.TaskRepo.CreateTask(task)
	if err != nil {
		return 0, err
	}

	return taskID, nil
}

// GetTask retrieves a task by its ID
func (s *TaskService) GetTask(taskID int64) (*models.Task, error) {
	task, err := s.TaskRepo.GetTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// GetUserTasks retrieves tasks by user ID
func (s *TaskService) GetUserTasks(userID int64) ([]*models.Task, error) {
	tasks, err := s.TaskRepo.GetTasksByUserID(userID)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		return nil, errors.WithStack(errors.Errorf("tasks for user ID %d do not exist", userID))
	}

	return tasks, nil
}
