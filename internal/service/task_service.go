package service

import (
	"time"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/queue"
	"github.com/pkg/errors"
)

// TaskService defines methods for task-related business logic
type TaskService struct {
	TaskRepo      repository.TaskRepository // TODO: case
	UserRepo      repository.UserRepository
	MessengerRepo repository.MessengerRepository
	producer      *queue.Producer
}

// NewTaskService creates a new TaskService
func NewTaskService(taskRepo repository.TaskRepository, userRepo repository.UserRepository, messengerRepo repository.MessengerRepository, producer *queue.Producer) *TaskService {
	return &TaskService{
		TaskRepo:      taskRepo,
		UserRepo:      userRepo,
		MessengerRepo: messengerRepo,
		producer:      producer,
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

// UpdateTask retrieves an existing task by its ID and updates it
func (s *TaskService) UpdateTask(taskID int64, updateRequest *models.TaskUpdateRequest) (*models.Task, error) {
	// Check if the task exists
	task, err := s.TaskRepo.GetTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	// Update the task fields (partial update)
	if updateRequest.Title != nil {
		task.Title = *updateRequest.Title
	}
	if updateRequest.Description != nil {
		task.Description = *updateRequest.Description
	}
	if updateRequest.Status != nil {
		task.Status = *updateRequest.Status
	}
	if updateRequest.DueDate != nil { // TODO: ...DueDate.IsZero()
		task.DueDate = *updateRequest.DueDate
	}

	// Save the updated task
	err = s.TaskRepo.UpdateTask(task)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// DeleteTask deletes a task by its ID (soft delete)
func (s *TaskService) DeleteTask(taskID int64) error {
	_, err := s.TaskRepo.GetTaskByID(taskID)
	if err != nil {
		return err
	}

	err = s.TaskRepo.DeleteTask(taskID)
	if err != nil {
		return err
	}

	return nil
}

// ScheduleTask sends a task to queue for interacting with scheduling service
func (s *TaskService) ScheduleTask(scheduledTask *models.ScheduledTask) error {
	// Check if the task exists
	task, err := s.TaskRepo.GetTaskByID(scheduledTask.TaskID)
	if err != nil {
		return err
	}

	if task == nil {
		return errors.WithStack(errors.Errorf("task with ID %d does not exist", scheduledTask.TaskID))
	}

	if task.DueDate.IsZero() {
		return errors.WithStack(errors.Errorf("task with ID %d has no DueDate value: it can't be nil", task.ID))
	}

	// messengerID, err := s.MessengerRepo.GetMessengerIDByName(scheduledTask.MessengerName)
	// if messengerID == 0 { // TODO: nil instead of 0
	// 	return errors.WithStack(errors.Errorf("messenger with name %s does not exist", scheduledTask.MessengerName))
	// }

	// s.MessengerRepo.GetMessengerRelatedUser() // TODO: resolve it somehow so that we dont have a need to pass ChatID in scheduledTask

	// Send the task to queue
	taskQueueMessage := map[string]interface{}{
		"task": scheduledTask.JobName,
		"args": []interface{}{scheduledTask.MessengerName, scheduledTask.ChatID, task.ID, task.Title, task.Description, task.DueDate},
	}

	err = s.producer.Publish(taskQueueMessage)
	if err != nil {
		// TODO: retry | failed to publish message: Exception (504) Reason: \"channel/connection is not open\"
		return errors.WithStack(errors.Errorf("can't publish message %v to rabbitmq: %s",
			taskQueueMessage,
			err,
		))
	}

	return nil
}
