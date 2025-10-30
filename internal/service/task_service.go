package service

import (
	"fmt"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/queue"

	"github.com/pkg/errors"
)

// TaskService defines methods for task-related business logic
type TaskService struct {
	taskRepo      repository.TaskRepository
	userRepo      repository.UserRepository
	messengerRepo repository.MessengerRepository
	producer      *queue.Producer
}

// NewTaskService creates a new TaskService
func NewTaskService(taskRepo repository.TaskRepository, userRepo repository.UserRepository, messengerRepo repository.MessengerRepository, producer *queue.Producer) *TaskService {
	return &TaskService{
		taskRepo:      taskRepo,
		userRepo:      userRepo,
		messengerRepo: messengerRepo,
		producer:      producer,
	}
}

// CreateTask implements BL of adding new task
func (s *TaskService) CreateTask(task *models.Task) (int64, error) {
	// check if user exists
	_, err := s.userRepo.GetUserByID(task.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}

		return 0, errors.WithStack(err)
	}

	if task.MessengerRelatedUserID != nil {

		// check if messenger related user exists
		_, err := s.messengerRepo.GetMessengerRelatedUserByID(*task.MessengerRelatedUserID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}

			return 0, errors.WithStack(err)
		}
	}

	taskID, err := s.taskRepo.CreateTask(task)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return taskID, nil
}

// GetTask implements BL of retrieving existing task by its id
func (s *TaskService) GetTask(taskID int64) (*models.Task, error) {
	task, err := s.taskRepo.GetTaskByID(taskID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return task, nil
}

// GetUserTasks implements BL of retrieving existing tasks by user id
func (s *TaskService) GetUserTasks(userID int64) ([]*models.Task, error) {
	// check if user exists
	_, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}

		return nil, errors.WithStack(err)
	}

	tasks, err := s.taskRepo.GetTasksByUserID(userID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return tasks, nil
}

// UpdateTask implements BL of updating task by id
func (s *TaskService) UpdateTask(taskID int64, updateRequest *models.TaskUpdateRequest) (*models.Task, error) {
	// check if the task exists
	task, err := s.taskRepo.GetTaskByID(taskID)
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
	if updateRequest.StartDate != nil {
		task.StartDate = *updateRequest.StartDate
	}
	if updateRequest.FinishDate != nil {
		task.FinishDate = updateRequest.FinishDate
	}
	// TODO: check if cron expression is valid
	if updateRequest.CronExpression != nil {
		task.CronExpression = updateRequest.CronExpression
	}

	err = s.taskRepo.UpdateTask(task)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return task, nil
}

// DeleteTask implements BL of soft deleting task by id
func (s *TaskService) DeleteTask(taskID int64) error {
	_, err := s.taskRepo.GetTaskByID(taskID)
	if err != nil {
		return errors.WithStack(err)
	}

	err = s.taskRepo.DeleteTask(taskID)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// QueueTask implements BL of sending task to queue for interacting with scheduler service
func (s *TaskService) QueueTask(scheduledTask *models.ScheduledTask) error {
	// check if task exists
	task, err := s.taskRepo.GetTaskByID(scheduledTask.TaskID)
	if err != nil {
		return errors.WithStack(err)
	}

	var taskQueueMessage map[string]interface{}
	// TODO: other actions
	if scheduledTask.Action == "schedule" {
		// if task.StartDate.IsZero() {
		// 	return errors.WithStack(errors.Errorf("task with ID %d has no StartDate value: it can't be nil", task.ID))
		// 	// 409
		// }
		// messengerID, err := s.messengerRepo.GetMessengerIDByName(scheduledTask.MessengerName)
		// if messengerID == 0 { // TODO: nil instead of 0
		// 	return errors.WithStack(errors.Errorf("messenger with name %s does not exist", scheduledTask.MessengerName))
		// }

		if task.MessengerRelatedUserID == nil {
			return errors.Wrap(errs.ErrUnprocessableEntity, fmt.Sprintf("task with ID %d has no MessengerRelatedUserID value", task.ID))
		}

		var messengerRelatedUser *models.MessengerRelatedUser

		// check if messenger related user indeed exists
		messengerRelatedUser, err = s.messengerRepo.GetMessengerRelatedUserByID(*task.MessengerRelatedUserID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}

			return errors.WithStack(err)
		}

		taskQueueMessage = map[string]interface{}{
			"task": "worker.schedule_task",
			"args": []interface{}{"telegram", messengerRelatedUser.ChatID, task.ID, task.Title, task.Description, task.StartDate, task.CronExpression},
		}

	} else {
		taskQueueMessage = map[string]interface{}{
			"task": "worker.delete_task",
			"args": []interface{}{task.ID, "telegram"},
		}
	}

	err = s.producer.Publish(taskQueueMessage)
	if err != nil {
		// TODO: failed to publish message: Exception (504) Reason: \"channel/connection is not open\"
		return errors.WithStack(errors.Errorf("can't publish message %v to rabbitmq: %s",
			taskQueueMessage,
			err,
		))
	}

	// TODO: log message has been published

	return nil
}
