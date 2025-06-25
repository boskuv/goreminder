package service

import (
	errs "github.com/boskuv/goreminder/internal/errors"
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

// CreateTask implements BL of adding new task
func (s *TaskService) CreateTask(task *models.Task) (int64, error) {
	// check if user exists
	_, err := s.UserRepo.GetUserByID(task.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}

		return 0, errors.WithStack(err)
	}

	if task.MessengerRelatedUserID != nil {

		// check if messenger related user exists
		_, err := s.MessengerRepo.GetUserID("")
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}

			return 0, errors.WithStack(err)
		}
	}

	taskID, err := s.TaskRepo.CreateTask(task)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return taskID, nil
}

// GetTask implements BL of retrieving existing task by its id
func (s *TaskService) GetTask(taskID int64) (*models.Task, error) {
	task, err := s.TaskRepo.GetTaskByID(taskID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return task, nil
}

// GetUserTasks implements BL of retrieving existing tasks by user id
func (s *TaskService) GetUserTasks(userID int64) ([]*models.Task, error) {
	// check if user exists
	_, err := s.UserRepo.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}

		return nil, errors.WithStack(err)
	}

	tasks, err := s.TaskRepo.GetTasksByUserID(userID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return tasks, nil
}

// UpdateTask implements BL of updating task by id
func (s *TaskService) UpdateTask(taskID int64, updateRequest *models.TaskUpdateRequest) (*models.Task, error) {
	// check if the task exists
	task, err := s.TaskRepo.GetTaskByID(taskID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// update the task fields (partial update)
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

	err = s.TaskRepo.UpdateTask(task)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return task, nil
}

// DeleteTask implements BL of soft deleting task by id
func (s *TaskService) DeleteTask(taskID int64) error {
	_, err := s.TaskRepo.GetTaskByID(taskID)
	if err != nil {
		return errors.WithStack(err)
	}

	err = s.TaskRepo.DeleteTask(taskID)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// ScheduleTask sends a task to queue for interacting with scheduling service
// func (s *TaskService) ScheduleTask(scheduledTask *models.ScheduledTask) error {
// 	// Check if the task exists
// 	task, err := s.TaskRepo.GetTaskByID(scheduledTask.TaskID)
// 	if err != nil {
// 		return err
// 	}

// 	if task == nil {
// 		return errors.WithStack(errors.Errorf("task with ID %d does not exist", scheduledTask.TaskID))
// 	}

// 	if task.DueDate.IsZero() {
// 		return errors.WithStack(errors.Errorf("task with ID %d has no DueDate value: it can't be nil", task.ID))
// 	}

// 	// messengerID, err := s.MessengerRepo.GetMessengerIDByName(scheduledTask.MessengerName)
// 	// if messengerID == 0 { // TODO: nil instead of 0
// 	// 	return errors.WithStack(errors.Errorf("messenger with name %s does not exist", scheduledTask.MessengerName))
// 	// }

// 	// s.MessengerRepo.GetMessengerRelatedUser() // TODO: resolve it somehow so that we dont have a need to pass ChatID in scheduledTask

// 	// Send the task to queue
// 	taskQueueMessage := map[string]interface{}{
// 		"task": scheduledTask.JobName,
// 		"args": []interface{}{scheduledTask.MessengerName, scheduledTask.ChatID, task.ID, task.Title, task.Description, task.DueDate},
// 	}

// 	err = s.producer.Publish(taskQueueMessage)
// 	if err != nil {
// 		// TODO: retry | failed to publish message: Exception (504) Reason: \"channel/connection is not open\"
// 		return errors.WithStack(errors.Errorf("can't publish message %v to rabbitmq: %s",
// 			taskQueueMessage,
// 			err,
// 		))
// 	}

// 	return nil
// }
