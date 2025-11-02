package service

import (
	"context"
	"fmt"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/queue"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pkg/errors"
)

// TaskService defines methods for task-related business logic
type TaskService struct {
	taskRepo        repository.TaskRepository
	userRepo        repository.UserRepository
	messengerRepo   repository.MessengerRepository
	taskHistoryRepo repository.TaskHistoryRepository
	producer        *queue.Producer
	tracer          trace.Tracer
}

// NewTaskService creates a new TaskService
func NewTaskService(taskRepo repository.TaskRepository, userRepo repository.UserRepository, messengerRepo repository.MessengerRepository, taskHistoryRepo repository.TaskHistoryRepository, producer *queue.Producer) *TaskService {
	return &TaskService{
		taskRepo:        taskRepo,
		userRepo:        userRepo,
		messengerRepo:   messengerRepo,
		taskHistoryRepo: taskHistoryRepo,
		producer:        producer,
		tracer:          otel.Tracer("task-service"),
	}
}

// CreateTask implements BL of adding new task
func (s *TaskService) CreateTask(ctx context.Context, task *models.Task) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.CreateTask",
		trace.WithAttributes(
			attribute.Int64("user.id", task.UserID),
		))
	defer span.End()

	// check if user exists
	_, err := s.userRepo.GetUserByID(ctx, task.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	if task.MessengerRelatedUserID != nil {
		span.SetAttributes(attribute.Int("messenger_related_user.id", *task.MessengerRelatedUserID))
		// check if messenger related user exists
		_, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *task.MessengerRelatedUserID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return 0, errors.WithStack(err)
		}
	}

	taskID, err := s.taskRepo.CreateTask(ctx, task)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("task.id", taskID))

	// Record history
	task.ID = taskID
	_, historySpan := s.tracer.Start(ctx, "task_service.record_task_created_history",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
			attribute.Int64("user.id", task.UserID),
		))
	history := &models.TaskHistory{
		TaskID:   taskID,
		UserID:   task.UserID,
		Action:   string(models.TaskHistoryActionCreated),
		NewValue: s.taskToMap(task),
	}
	if err := s.taskHistoryRepo.CreateTaskHistory(ctx, history); err != nil {
		historySpan.RecordError(err)
		historySpan.SetStatus(codes.Error, err.Error())
		// Log error but don't fail the creation
		// TODO: log error
	} else {
		historySpan.SetStatus(codes.Ok, "history recorded")
	}
	historySpan.End()

	span.SetStatus(codes.Ok, "task created successfully")
	return taskID, nil
}

// GetTask implements BL of retrieving existing task by its id
func (s *TaskService) GetTask(ctx context.Context, taskID int64) (*models.Task, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.GetTask",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
		))
	defer span.End()

	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	span.SetStatus(codes.Ok, "task retrieved successfully")
	return task, nil
}

// GetUserTasks implements BL of retrieving existing tasks by user id
func (s *TaskService) GetUserTasks(ctx context.Context, userID int64) ([]*models.Task, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.GetUserTasks",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
		))
	defer span.End()

	// check if user exists
	_, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	tasks, err := s.taskRepo.GetTasksByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int("tasks.count", len(tasks)))
	span.SetStatus(codes.Ok, "user tasks retrieved successfully")
	return tasks, nil
}

// UpdateTask implements BL of updating task by id
func (s *TaskService) UpdateTask(ctx context.Context, taskID int64, updateRequest *models.TaskUpdateRequest) (*models.Task, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.UpdateTask",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
		))
	defer span.End()

	// check if the task exists
	oldTask, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Create a copy for old values
	oldTaskMap := s.taskToMap(oldTask)
	oldStatus := oldTask.Status
	statusChanged := false

	// update the task fields (partial update)
	if updateRequest.Title != nil {
		oldTask.Title = *updateRequest.Title
	}
	if updateRequest.Description != nil {
		oldTask.Description = *updateRequest.Description
	}
	if updateRequest.Status != nil {
		if oldTask.Status != *updateRequest.Status {
			statusChanged = true
		}
		oldTask.Status = *updateRequest.Status
	}
	if updateRequest.StartDate != nil {
		oldTask.StartDate = *updateRequest.StartDate
	}
	if updateRequest.FinishDate != nil {
		oldTask.FinishDate = updateRequest.FinishDate
	}
	// TODO: check if cron expression is valid
	if updateRequest.CronExpression != nil {
		oldTask.CronExpression = updateRequest.CronExpression
	}

	err = s.taskRepo.UpdateTask(ctx, oldTask)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Record status change separately if status was changed
	if statusChanged {
		_, statusHistorySpan := s.tracer.Start(ctx, "task_service.record_status_changed_history",
			trace.WithAttributes(
				attribute.Int64("task.id", taskID),
				attribute.String("status.old", oldStatus),
				attribute.String("status.new", oldTask.Status),
			))
		statusHistory := &models.TaskHistory{
			TaskID:   taskID,
			UserID:   oldTask.UserID,
			Action:   string(models.TaskHistoryActionStatusChanged),
			OldValue: map[string]interface{}{"status": oldStatus},
			NewValue: map[string]interface{}{"status": oldTask.Status},
		}
		if err := s.taskHistoryRepo.CreateTaskHistory(ctx, statusHistory); err != nil {
			statusHistorySpan.RecordError(err)
			statusHistorySpan.SetStatus(codes.Error, err.Error())
			// TODO: log error
		} else {
			statusHistorySpan.SetStatus(codes.Ok, "status change history recorded")
		}
		statusHistorySpan.End()
	}

	// Record general update history (if other fields changed, not just status)
	hasOtherChanges := updateRequest.Title != nil || updateRequest.Description != nil ||
		updateRequest.StartDate != nil || updateRequest.FinishDate != nil ||
		updateRequest.CronExpression != nil

	if hasOtherChanges || (updateRequest.Status != nil && !statusChanged) {
		_, updateHistorySpan := s.tracer.Start(ctx, "task_service.record_task_updated_history",
			trace.WithAttributes(
				attribute.Int64("task.id", taskID),
				attribute.Int64("user.id", oldTask.UserID),
			))
		history := &models.TaskHistory{
			TaskID:   taskID,
			UserID:   oldTask.UserID,
			Action:   string(models.TaskHistoryActionUpdated),
			OldValue: oldTaskMap,
			NewValue: s.taskToMap(oldTask),
		}
		if err := s.taskHistoryRepo.CreateTaskHistory(ctx, history); err != nil {
			updateHistorySpan.RecordError(err)
			updateHistorySpan.SetStatus(codes.Error, err.Error())
			// TODO: log error
		} else {
			updateHistorySpan.SetStatus(codes.Ok, "update history recorded")
		}
		updateHistorySpan.End()
	}

	span.SetStatus(codes.Ok, "task updated successfully")
	return oldTask, nil
}

// DeleteTask implements BL of soft deleting task by id
func (s *TaskService) DeleteTask(ctx context.Context, taskID int64) error {
	ctx, span := s.tracer.Start(ctx, "task_service.DeleteTask",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
		))
	defer span.End()

	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	err = s.taskRepo.DeleteTask(ctx, taskID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	// Record history
	_, deleteHistorySpan := s.tracer.Start(ctx, "task_service.record_task_deleted_history",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
			attribute.Int64("user.id", task.UserID),
		))
	history := &models.TaskHistory{
		TaskID:   taskID,
		UserID:   task.UserID,
		Action:   string(models.TaskHistoryActionDeleted),
		OldValue: s.taskToMap(task),
	}
	if err := s.taskHistoryRepo.CreateTaskHistory(ctx, history); err != nil {
		deleteHistorySpan.RecordError(err)
		deleteHistorySpan.SetStatus(codes.Error, err.Error())
		// TODO: log error
	} else {
		deleteHistorySpan.SetStatus(codes.Ok, "delete history recorded")
	}
	deleteHistorySpan.End()

	span.SetStatus(codes.Ok, "task deleted successfully")
	return nil
}

// QueueTask implements BL of sending task to queue for interacting with scheduler service
func (s *TaskService) QueueTask(ctx context.Context, scheduledTask *models.ScheduledTask) error {
	ctx, span := s.tracer.Start(ctx, "task_service.QueueTask",
		trace.WithAttributes(
			attribute.Int64("task.id", scheduledTask.TaskID),
			attribute.String("action", scheduledTask.Action),
		))
	defer span.End()

	// check if task exists
	task, err := s.taskRepo.GetTaskByID(ctx, scheduledTask.TaskID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
			err := errors.Wrap(errs.ErrUnprocessableEntity, fmt.Sprintf("task with ID %d has no MessengerRelatedUserID value", task.ID))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		var messengerRelatedUser *models.MessengerRelatedUser

		// check if messenger related user indeed exists
		messengerRelatedUser, err = s.messengerRepo.GetMessengerRelatedUserByID(ctx, *task.MessengerRelatedUserID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
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

	err = s.producer.Publish(ctx, taskQueueMessage)
	if err != nil {
		// TODO: failed to publish message: Exception (504) Reason: \"channel/connection is not open\"
		err = errors.Errorf("can't publish message %v to rabbitmq: %s",
			taskQueueMessage,
			err,
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	span.SetStatus(codes.Ok, "task queued successfully")
	return nil
}

// GetTaskHistory implements BL of retrieving task history by task ID
func (s *TaskService) GetTaskHistory(ctx context.Context, taskID int64) ([]*models.TaskHistory, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.GetTaskHistory",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
		))
	defer span.End()

	// Check if task exists
	_, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	histories, err := s.taskHistoryRepo.GetTaskHistoryByTaskID(ctx, taskID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int("history.count", len(histories)))
	span.SetStatus(codes.Ok, "task history retrieved successfully")
	return histories, nil
}

// GetUserTaskHistory implements BL of retrieving task history by user ID
func (s *TaskService) GetUserTaskHistory(ctx context.Context, userID int64, limit, offset int) ([]*models.TaskHistory, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.GetUserTaskHistory",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
			attribute.Int("limit", limit),
			attribute.Int("offset", offset),
		))
	defer span.End()

	// Check if user exists
	_, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	histories, err := s.taskHistoryRepo.GetTaskHistoryByUserID(ctx, userID, limit, offset)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int("history.count", len(histories)))
	span.SetStatus(codes.Ok, "user task history retrieved successfully")
	return histories, nil
}

// taskToMap converts a task to a map for history storage
func (s *TaskService) taskToMap(task *models.Task) map[string]interface{} {
	result := map[string]interface{}{
		"id":          task.ID,
		"title":       task.Title,
		"description": task.Description,
		"status":      task.Status,
	}

	if !task.StartDate.IsZero() {
		result["start_date"] = task.StartDate
	}
	if task.FinishDate != nil {
		result["finish_date"] = *task.FinishDate
	}
	if task.CronExpression != nil {
		result["cron_expression"] = *task.CronExpression
	}
	if task.MessengerRelatedUserID != nil {
		result["messenger_related_user_id"] = *task.MessengerRelatedUserID
	}

	return result
}
