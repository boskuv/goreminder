package service

import (
	"context"
	"fmt"
	"time"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/boskuv/goreminder/pkg/queue"
	"github.com/gorhill/cronexpr"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// TaskService defines methods for task-related business logic
type TaskService struct {
	taskRepo        repository.TaskRepository
	userRepo        repository.UserRepository
	messengerRepo   repository.MessengerRepository
	taskHistoryRepo repository.TaskHistoryRepository
	producer        *queue.Producer
	tracer          trace.Tracer
	logger          zerolog.Logger
}

// NewTaskService creates a new TaskService
func NewTaskService(taskRepo repository.TaskRepository, userRepo repository.UserRepository, messengerRepo repository.MessengerRepository, taskHistoryRepo repository.TaskHistoryRepository, producer *queue.Producer, logger zerolog.Logger) *TaskService {
	return &TaskService{
		taskRepo:        taskRepo,
		userRepo:        userRepo,
		messengerRepo:   messengerRepo,
		taskHistoryRepo: taskHistoryRepo,
		producer:        producer,
		tracer:          otel.Tracer("task-service"),
		logger:          logger,
	}
}

// CreateTask implements BL of adding new task
func (s *TaskService) CreateTask(ctx context.Context, task *models.Task) (int64, int64, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.CreateTask",
		trace.WithAttributes(
			attribute.Int64("user.id", task.UserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", task.UserID).
		Str("task.title", task.Title).
		Msg("starting task creation")

	// check if user exists
	_, err := s.userRepo.GetUserByID(ctx, task.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		log.Debug().
			Err(err).
			Int64("user.id", task.UserID).
			Msg("user not found or error retrieving user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, 0, errors.WithStack(err)
	}
	log.Debug().
		Int64("user.id", task.UserID).
		Msg("user exists, proceeding with task creation")

	// Validate and set default status
	if task.Status == "" {
		task.Status = string(models.TaskStatusPending)
	} else {
		if err := models.ValidateTaskStatus(task.Status); err != nil {
			log.Debug().
				Err(err).
				Str("status", task.Status).
				Msg("invalid task status")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return 0, 0, errors.Wrap(errs.ErrValidation, err.Error())
		}
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
			return 0, 0, errors.WithStack(err)
		}
	}

	log.Debug().
		Int64("user.id", task.UserID).
		Msg("creating task in repository")
	taskID, err := s.taskRepo.CreateTask(ctx, task)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", task.UserID).
			Msg("failed to create task in repository")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, 0, errors.WithStack(err)
	}

	span.SetAttributes(attribute.Int64("task.id", taskID))
	log.Debug().
		Int64("task.id", taskID).
		Int64("user.id", task.UserID).
		Msg("task created in repository, recording history")

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
		log.Debug().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to record task history")
		historySpan.RecordError(err)
		historySpan.SetStatus(codes.Error, err.Error())
	} else {
		log.Debug().
			Int64("task.id", taskID).
			Msg("task history recorded successfully")
		historySpan.SetStatus(codes.Ok, "history recorded")
	}
	historySpan.End()

	// If task has cron_expression and requires_confirmation, create a child task
	var childTaskID int64
	if task.CronExpression != nil && task.RequiresConfirmation {
		log.Debug().
			Int64("task.id", taskID).
			Str("cron_expression", *task.CronExpression).
			Msg("creating child task for cron task with confirmation")

		// Create child task
		childTask := &models.Task{
			Title:                  task.Title,
			Description:            task.Description,
			UserID:                 task.UserID,
			MessengerRelatedUserID: task.MessengerRelatedUserID,
			ParentID:               &taskID,
			StartDate:              task.StartDate,
			FinishDate:             task.FinishDate,
			CronExpression:         nil, // Child tasks don't have cron expression
			RequiresConfirmation:   task.RequiresConfirmation,
			Status:                 string(models.TaskStatusPending),
		}

		childTaskID, err = s.taskRepo.CreateTask(ctx, childTask)
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", taskID).
				Msg("failed to create child task")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			// Don't fail the main task creation, just log the error
		} else {
			log.Debug().
				Int64("task.id", taskID).
				Int64("child_task.id", childTaskID).
				Time("child_start_date", task.StartDate).
				Msg("child task created successfully")
			span.SetAttributes(attribute.Int64("child_task.id", childTaskID))
		}
	}

	log.Debug().
		Int64("task.id", taskID).
		Int64("user.id", task.UserID).
		Msg("task creation completed successfully")
	span.SetStatus(codes.Ok, "task created successfully")
	return taskID, childTaskID, nil
}

// GetTask implements BL of retrieving existing task by its id
func (s *TaskService) GetTask(ctx context.Context, taskID int64) (*models.Task, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.GetTask",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("task.id", taskID).
		Msg("getting task")

	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to get task")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("task.id", taskID).
		Msg("task retrieved successfully")
	span.SetStatus(codes.Ok, "task retrieved successfully")
	return task, nil
}

// GetUserTasks implements BL of retrieving existing tasks by user id with pagination and ordering
func (s *TaskService) GetUserTasks(ctx context.Context, userID int64, page, pageSize int, orderBy string, startDateFrom, startDateTo, createdAtFrom, createdAtTo *time.Time, requiresConfirmation *bool) ([]*models.Task, int, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.GetUserTasks",
		trace.WithAttributes(
			attribute.Int64("user.id", userID),
			attribute.Int("page", page),
			attribute.Int("page_size", pageSize),
			attribute.String("order_by", orderBy),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", userID).
		Int("page", page).
		Int("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting user tasks")

	// check if user exists
	_, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("user not found when getting tasks")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.WithStack(err)
	}

	tasks, totalCount, err := s.taskRepo.GetTasksByUserIDWithPagination(ctx, userID, page, pageSize, orderBy, startDateFrom, startDateTo, createdAtFrom, createdAtTo, requiresConfirmation)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get user tasks")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.WithStack(err)
	}

	log.Debug().
		Int64("user.id", userID).
		Int("tasks.count", len(tasks)).
		Int("total_count", totalCount).
		Msg("user tasks retrieved successfully")
	span.SetAttributes(attribute.Int("tasks.count", len(tasks)))
	span.SetAttributes(attribute.Int("total_count", totalCount))
	span.SetStatus(codes.Ok, "user tasks retrieved successfully")
	return tasks, totalCount, nil
}

// UpdateTask implements BL of updating task by id
func (s *TaskService) UpdateTask(ctx context.Context, taskID int64, updateRequest *models.TaskUpdateRequest) (*models.Task, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.UpdateTask",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("task.id", taskID).
		Msg("updating task")

	// check if the task exists
	oldTask, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to get task for update")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Create a copy for old values
	oldTaskMap := s.taskToMap(oldTask)
	oldStatus := oldTask.Status
	statusChanged := false

	// Save old values for comparison (before updating)
	oldTitle := oldTask.Title
	oldDescription := oldTask.Description
	oldStartDate := oldTask.StartDate
	oldCronExpression := oldTask.CronExpression
	oldRequiresConfirmation := oldTask.RequiresConfirmation
	oldFinishDate := oldTask.FinishDate

	// Check if this is a parent task (has cron_expression)
	isParentTask := oldTask.CronExpression != nil

	// update the task fields (partial update)
	if updateRequest.Title != nil {
		oldTask.Title = *updateRequest.Title
	}
	if updateRequest.Description != nil {
		oldTask.Description = *updateRequest.Description
	}
	if updateRequest.Status != nil {
		// Validate status
		if err := models.ValidateTaskStatus(*updateRequest.Status); err != nil {
			log.Debug().
				Err(err).
				Str("status", *updateRequest.Status).
				Msg("invalid task status in update")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.Wrap(errs.ErrValidation, err.Error())
		}
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
	// TODO: check if cron expression is valid -> remove UpdateModel -> TaskModel
	if updateRequest.CronExpression != nil {
		oldTask.CronExpression = updateRequest.CronExpression
	}

	if updateRequest.RequiresConfirmation != nil {
		oldTask.RequiresConfirmation = *updateRequest.RequiresConfirmation
	}

	// Get database connection for transaction if we need to update child tasks
	var db *sqlx.DB
	var tx *sqlx.Tx
	var shouldRollback = false
	if isParentTask {
		db = s.taskRepo.GetDB()
		if db == nil {
			err := errors.New("database connection not available")
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", taskID).
				Msg("failed to get database connection for transaction")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}

		// Begin transaction for child tasks updates
		tx, err = db.BeginTxx(ctx, nil)
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", taskID).
				Msg("failed to begin transaction for child tasks update")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.Wrap(err, "failed to begin transaction")
		}

		shouldRollback = true
		defer func() {
			if shouldRollback {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					log.Error().
						Stack().
						Err(rollbackErr).
						Int64("task.id", taskID).
						Msg("failed to rollback transaction")
				}
			}
		}()
	}

	// Update parent task first
	err = s.taskRepo.UpdateTask(ctx, oldTask)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to update task")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Handle queue publishing for single tasks (without cron) or parent tasks when status changes
	// For parent tasks, we only publish if status changes to deleted (parent tasks don't execute directly)
	// For single tasks, we publish schedule_task or delete_task based on changes
	if !isParentTask {
		// For single tasks (without cron), publish to queue based on changes
		titleChanged := updateRequest.Title != nil && *updateRequest.Title != oldTitle
		descriptionChanged := updateRequest.Description != nil && *updateRequest.Description != oldDescription
		startDateChanged := updateRequest.StartDate != nil && !updateRequest.StartDate.Equal(oldStartDate)
		statusChangedToDeleted := statusChanged && oldTask.Status == string(models.TaskStatusDeleted)
		statusChangedToScheduled := statusChanged && oldTask.Status == string(models.TaskStatusScheduled)

		// Publish delete_task if status changed to deleted
		if statusChangedToDeleted {
			if oldTask.MessengerRelatedUserID != nil {
				taskQueueMessage := map[string]interface{}{
					"task": "worker.delete_task",
					"args": []interface{}{oldTask.ID, "telegram"},
				}

				err = s.producer.Publish(ctx, taskQueueMessage)
				if err != nil {
					log.Error().
						Stack().
						Err(err).
						Int64("task.id", taskID).
						Msg("failed to queue delete_task message for task with deleted status")
					// Don't fail the operation, just log the error
					// The database update was successful, queue update failure is non-critical
				} else {
					log.Debug().
						Int64("task.id", taskID).
						Msg("delete_task message queued successfully for deleted task")
				}
			}
		} else if statusChangedToScheduled || startDateChanged || titleChanged || descriptionChanged {
			// Publish schedule_task if status changed to scheduled or relevant fields changed
			if oldTask.MessengerRelatedUserID != nil {
				messengerRelatedUser, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *oldTask.MessengerRelatedUserID)
				if err != nil {
					log.Error().
						Stack().
						Err(err).
						Int64("task.id", taskID).
						Msg("failed to get messenger related user for task queue update")
					// Don't fail the operation, just log the error
				} else {
					taskQueueMessage := map[string]interface{}{
						"task": "worker.schedule_task",
						"args": []interface{}{"telegram", messengerRelatedUser.ChatID, oldTask.ID, oldTask.Title, oldTask.Description, oldTask.StartDate, oldTask.CronExpression, oldTask.RequiresConfirmation},
					}

					err = s.producer.Publish(ctx, taskQueueMessage)
					if err != nil {
						log.Error().
							Stack().
							Err(err).
							Int64("task.id", taskID).
							Msg("failed to queue schedule_task message for updated task")
						// Don't fail the operation, just log the error
						// The database update was successful, queue update failure is non-critical
					} else {
						log.Debug().
							Int64("task.id", taskID).
							Msg("schedule_task message queued successfully for updated task")
					}
				}
			}
		}
	} else if statusChanged {
		// For parent tasks, only publish if status changed to deleted
		// Parent tasks don't execute directly, so we don't publish schedule_task for them
		statusChangedToDeleted := oldTask.Status == string(models.TaskStatusDeleted)
		if statusChangedToDeleted {
			if oldTask.MessengerRelatedUserID != nil {
				taskQueueMessage := map[string]interface{}{
					"task": "worker.delete_task",
					"args": []interface{}{oldTask.ID, "telegram"},
				}

				err = s.producer.Publish(ctx, taskQueueMessage)
				if err != nil {
					log.Error().
						Stack().
						Err(err).
						Int64("task.id", taskID).
						Msg("failed to queue delete_task message for parent task with deleted status")
					// Don't fail the operation, just log the error
					// The database update was successful, queue update failure is non-critical
				} else {
					log.Debug().
						Int64("task.id", taskID).
						Msg("delete_task message queued successfully for deleted parent task")
				}
			}
		}
	}

	// Handle child tasks synchronization if this is a parent task
	if isParentTask {
		// Get all child tasks
		childTasks, err := s.taskRepo.GetChildTasksByParentID(ctx, taskID)
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", taskID).
				Msg("failed to get child tasks for synchronization")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.WithStack(err)
		}

		// Check if requires_confirmation was removed (true -> false)
		requiresConfirmationRemoved := oldRequiresConfirmation && !oldTask.RequiresConfirmation

		if requiresConfirmationRemoved {
			// Delete all child tasks from queue and database
			log.Debug().
				Int64("task.id", taskID).
				Msg("requires_confirmation removed, deleting all child tasks")

			if len(childTasks) > 0 {
				// Delete all child tasks in transaction
				err = s.taskRepo.DeleteChildTasksWithTx(ctx, tx, taskID)
				if err != nil {
					log.Error().
						Stack().
						Err(err).
						Int64("task.id", taskID).
						Msg("failed to delete child tasks in transaction")
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return nil, errors.WithStack(err)
				}

				// Queue delete_task message for each child task
				for _, childTask := range childTasks {
					childTaskQueueMessage := map[string]interface{}{
						"task": "worker.delete_task",
						"args": []interface{}{childTask.ID, "telegram"},
					}

					err = s.producer.Publish(ctx, childTaskQueueMessage)
					if err != nil {
						log.Error().
							Stack().
							Err(err).
							Int64("task.id", taskID).
							Int64("child_task.id", childTask.ID).
							Msg("failed to queue delete_task message for child task, rolling back transaction")
						span.RecordError(err)
						span.SetStatus(codes.Error, err.Error())
						return nil, errors.Wrap(err, "failed to queue delete_task message for child task")
					}
				}

				log.Debug().
					Int64("task.id", taskID).
					Int("child_tasks.count", len(childTasks)).
					Msg("child tasks deleted and queued successfully")
			}
		} else if len(childTasks) > 0 {
			// Update child tasks based on parent changes
			titleChanged := updateRequest.Title != nil && *updateRequest.Title != oldTitle
			descriptionChanged := updateRequest.Description != nil && *updateRequest.Description != oldDescription
			startDateChanged := updateRequest.StartDate != nil && !updateRequest.StartDate.Equal(oldStartDate)
			cronExpressionChanged := (updateRequest.CronExpression != nil && oldCronExpression == nil) ||
				(updateRequest.CronExpression == nil && oldCronExpression != nil) ||
				(updateRequest.CronExpression != nil && oldCronExpression != nil && *updateRequest.CronExpression != *oldCronExpression)
			finishDateChanged := (updateRequest.FinishDate != nil && oldFinishDate == nil) ||
				(updateRequest.FinishDate == nil && oldFinishDate != nil) ||
				(updateRequest.FinishDate != nil && oldFinishDate != nil && !updateRequest.FinishDate.Equal(*oldFinishDate))

			// Update each child task (skip done/deleted tasks)
			for _, childTask := range childTasks {
				// Skip already done or deleted tasks
				if childTask.Status == string(models.TaskStatusDone) {
					continue
				}

				childUpdated := false
				startDateUpdated := false

				// Update title if changed
				if titleChanged {
					childTask.Title = oldTask.Title
					childUpdated = true
				}

				// Update description if changed
				if descriptionChanged {
					childTask.Description = oldTask.Description
					childUpdated = true
				}

				// Update finish_date if changed
				if finishDateChanged {
					childTask.FinishDate = oldTask.FinishDate
					childUpdated = true
				}

				// Recalculate start_date if cron_expression or start_date changed
				if cronExpressionChanged || startDateChanged {
					if oldTask.CronExpression != nil {
						// Calculate next execution time from new cron expression
						// Use current time or the new start_date as base
						// If startDate has already passed, use time.Now().UTC() instead
						baseTime := time.Now().UTC()
						if startDateChanged {
							// Check if the new startDate has already passed
							if oldTask.StartDate.After(time.Now().UTC()) {
								baseTime = oldTask.StartDate
							} else {
								// startDate has already passed, use current time
								baseTime = time.Now().UTC()
							}
						} else if !childTask.StartDate.IsZero() {
							// Check if childTask.StartDate has already passed, use current time
							if childTask.StartDate.Before(time.Now().UTC()) {
								baseTime = time.Now().UTC()
							}
						}
						nextTime := cronexpr.MustParse(*oldTask.CronExpression).Next(baseTime)
						childTask.StartDate = nextTime
						childUpdated = true
						startDateUpdated = true

						log.Debug().
							Int64("task.id", taskID).
							Int64("child_task.id", childTask.ID).
							Time("new_start_date", nextTime).
							Time("base_time", baseTime).
							Msg("recalculated child task start_date from cron expression")
					} else if startDateChanged {
						// If cron expression was removed but start_date changed, update start_date
						childTask.StartDate = oldTask.StartDate
						childUpdated = true
						startDateUpdated = true
					}
				}

				// Update child task if any field changed
				if childUpdated {
					err = s.taskRepo.UpdateTaskWithTx(ctx, tx, childTask)
					if err != nil {
						log.Error().
							Stack().
							Err(err).
							Int64("task.id", taskID).
							Int64("child_task.id", childTask.ID).
							Msg("failed to update child task in transaction")
						span.RecordError(err)
						span.SetStatus(codes.Error, err.Error())
						return nil, errors.WithStack(err)
					}

					// Publish update to queue if start_date, title, or description changed
					// This is needed to update the task in scheduler
					if startDateUpdated || titleChanged || descriptionChanged {
						if childTask.MessengerRelatedUserID != nil {
							messengerRelatedUser, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *childTask.MessengerRelatedUserID)
							if err != nil {
								log.Error().
									Stack().
									Err(err).
									Int64("task.id", taskID).
									Int64("child_task.id", childTask.ID).
									Msg("failed to get messenger related user for child task queue update")
								// Don't fail the operation, just log the error
							} else {
								childTaskQueueMessage := map[string]interface{}{
									"task": "worker.schedule_task",
									"args": []interface{}{"telegram", messengerRelatedUser.ChatID, childTask.ID, childTask.Title, childTask.Description, childTask.StartDate, childTask.CronExpression, childTask.RequiresConfirmation},
								}

								err = s.producer.Publish(ctx, childTaskQueueMessage)
								if err != nil {
									log.Error().
										Stack().
										Err(err).
										Int64("task.id", taskID).
										Int64("child_task.id", childTask.ID).
										Msg("failed to queue schedule_task message for updated child task")
									// Don't fail the operation, just log the error
									// The database update was successful, queue update failure is non-critical
								} else {
									log.Debug().
										Int64("task.id", taskID).
										Int64("child_task.id", childTask.ID).
										Msg("child task update queued successfully")
								}
							}
						}
					}
				}
			}

			log.Debug().
				Int64("task.id", taskID).
				Int("child_tasks.count", len(childTasks)).
				Msg("child tasks synchronized successfully")
		}

		// Commit transaction if we started one
		if shouldRollback {
			err = tx.Commit()
			if err != nil {
				log.Error().
					Stack().
					Err(err).
					Int64("task.id", taskID).
					Msg("failed to commit transaction")
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, errors.Wrap(err, "failed to commit transaction")
			}
			shouldRollback = false
		}
	}

	// If requires_confirmation was added (false -> true) and task has cron_expression, create child tasks
	if !oldRequiresConfirmation && oldTask.RequiresConfirmation && oldTask.CronExpression != nil {
		log.Debug().
			Int64("task.id", taskID).
			Str("cron_expression", *oldTask.CronExpression).
			Msg("requires_confirmation added, creating child tasks")

		// Calculate next execution time from cron expression
		nextTime := cronexpr.MustParse(*oldTask.CronExpression).Next(time.Now().UTC())

		// Create child task
		childTask := &models.Task{
			Title:                  oldTask.Title,
			Description:            oldTask.Description,
			UserID:                 oldTask.UserID,
			MessengerRelatedUserID: oldTask.MessengerRelatedUserID,
			ParentID:               &taskID,
			StartDate:              nextTime,
			FinishDate:             oldTask.FinishDate,
			CronExpression:         nil, // Child tasks don't have cron expression
			RequiresConfirmation:   oldTask.RequiresConfirmation,
			Status:                 string(models.TaskStatusPending),
		}

		childTaskID, err := s.taskRepo.CreateTask(ctx, childTask)
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", taskID).
				Msg("failed to create child task after adding requires_confirmation")
			// Don't fail the operation, just log the error
		} else {
			log.Debug().
				Int64("task.id", taskID).
				Int64("child_task.id", childTaskID).
				Time("child_start_date", nextTime).
				Msg("child task created successfully after adding requires_confirmation")
			span.SetAttributes(attribute.Int64("child_task.id", childTaskID))
		}
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
		updateRequest.CronExpression != nil || updateRequest.RequiresConfirmation != nil

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

	log.Debug().
		Int64("task.id", taskID).
		Msg("task updated successfully")
	span.SetStatus(codes.Ok, "task updated successfully")
	return oldTask, nil
}

// DeleteTask implements BL of soft deleting task by id
// If queueing fails, the database update is rolled back
func (s *TaskService) DeleteTask(ctx context.Context, taskID int64) error {
	ctx, span := s.tracer.Start(ctx, "task_service.DeleteTask",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("task.id", taskID).
		Msg("deleting task")

	task, err := s.taskRepo.GetTaskByIDWithoutStatusFilter(ctx, taskID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to get task for deletion")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	// Get database connection for transaction
	db := s.taskRepo.GetDB()
	if db == nil {
		err := errors.New("database connection not available")
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to get database connection for transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	// Begin transaction
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to begin transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to begin transaction")
	}

	// Track if we need to rollback
	var shouldRollback = true
	defer func() {
		if shouldRollback {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Error().
					Stack().
					Err(rollbackErr).
					Int64("task.id", taskID).
					Msg("failed to rollback transaction")
			}
		}
	}()

	// If task has cron_expression, it's a parent task - delete all child tasks first
	if task.CronExpression != nil {
		log.Debug().
			Int64("task.id", taskID).
			Msg("deleting child tasks for parent task")

		// Get all child tasks
		childTasks, err := s.taskRepo.GetChildTasksByParentID(ctx, taskID)
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", taskID).
				Msg("failed to get child tasks for deletion")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return errors.WithStack(err)
		}

		if len(childTasks) > 0 {
			// Delete all child tasks in transaction
			err = s.taskRepo.DeleteChildTasksWithTx(ctx, tx, taskID)
			if err != nil {
				log.Error().
					Stack().
					Err(err).
					Int64("task.id", taskID).
					Msg("failed to delete child tasks in transaction")
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return errors.WithStack(err)
			}

			// Queue delete_task message for each child task
			// If this fails, we'll rollback the transaction
			for _, childTask := range childTasks {
				childTaskQueueMessage := map[string]interface{}{
					"task": "worker.delete_task",
					"args": []interface{}{childTask.ID, "telegram"},
				}

				err = s.producer.Publish(ctx, childTaskQueueMessage)
				if err != nil {
					log.Error().
						Stack().
						Err(err).
						Int64("task.id", taskID).
						Int64("child_task.id", childTask.ID).
						Msg("failed to queue delete_task message for child task, rolling back transaction")
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					// Transaction will be rolled back in defer
					return errors.Wrap(err, "failed to queue delete_task message for child task")
				}
			}

			log.Debug().
				Int64("task.id", taskID).
				Int("child_tasks.count", len(childTasks)).
				Msg("child tasks deleted and queued successfully")
			span.SetAttributes(attribute.Int("child_tasks.count", len(childTasks)))
		}
	}

	// Delete the task in transaction
	err = s.taskRepo.DeleteTaskWithTx(ctx, tx, taskID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to delete task in transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	// Queue delete_task message for the task
	// If this fails, we'll rollback the transaction
	taskQueueMessage := map[string]interface{}{
		"task": "worker.delete_task",
		"args": []interface{}{task.ID, "telegram"},
	}

	err = s.producer.Publish(ctx, taskQueueMessage)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to queue delete_task message, rolling back transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Transaction will be rolled back in defer
		return errors.Wrap(err, "failed to queue delete_task message")
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to commit transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to commit transaction")
	}

	// Mark that we've committed, so defer won't rollback
	shouldRollback = false

	// Record history (outside transaction, as it's not critical if it fails)
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
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to record task deleted history")
		deleteHistorySpan.RecordError(err)
		deleteHistorySpan.SetStatus(codes.Error, err.Error())
	} else {
		deleteHistorySpan.SetStatus(codes.Ok, "delete history recorded")
	}
	deleteHistorySpan.End()

	log.Debug().
		Int64("task.id", taskID).
		Msg("task deleted successfully")
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

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("task.id", scheduledTask.TaskID).
		Str("action", scheduledTask.Action).
		Msg("queuing task")

	// check if task exists
	task, err := s.taskRepo.GetTaskByID(ctx, scheduledTask.TaskID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", scheduledTask.TaskID).
			Msg("failed to get task for queuing")
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
			"args": []interface{}{"telegram", messengerRelatedUser.ChatID, task.ID, task.Title, task.Description, task.StartDate, task.CronExpression, task.RequiresConfirmation},
		}

	} else {
		taskQueueMessage = map[string]interface{}{
			"task": "worker.delete_task",
			"args": []interface{}{task.ID, "telegram"},
		}
	}

	err = s.producer.Publish(ctx, taskQueueMessage)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", scheduledTask.TaskID).
			Str("action", scheduledTask.Action).
			Msg("failed to queue task")
		// TODO: failed to publish message: Exception (504) Reason: \"channel/connection is not open\"
		err = errors.Errorf("can't publish message %v to rabbitmq: %s",
			taskQueueMessage,
			err,
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	log.Debug().
		Int64("task.id", scheduledTask.TaskID).
		Str("action", scheduledTask.Action).
		Msg("task queued successfully")
	span.SetStatus(codes.Ok, "task queued successfully")
	return nil
}

// MarkTaskAsDone marks a task as done and queues worker.delete_task in a transactional manner
// If queueing fails, the database update is rolled back
func (s *TaskService) MarkTaskAsDone(ctx context.Context, taskID int64) (*models.Task, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.MarkTaskAsDone",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("task.id", taskID).
		Msg("marking task as done")

	// Check if the task exists (without status filter to allow checking already-done tasks)
	task, err := s.taskRepo.GetTaskByIDWithoutStatusFilter(ctx, taskID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to get task for marking as done")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Check if task is already done (idempotent operation)
	if task.Status == string(models.TaskStatusDone) {
		log.Debug().
			Int64("task.id", taskID).
			Msg("task is already marked as done")
		// Return the task as-is, consider it idempotent
		return task, nil
	}

	// Store old status for history
	oldStatus := task.Status

	// Get database connection for transaction
	db := s.taskRepo.GetDB()
	if db == nil {
		err := errors.New("database connection not available")
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to get database connection for transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Begin transaction
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to begin transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to begin transaction")
	}

	// Track if we need to rollback
	var shouldRollback = true
	defer func() {
		if shouldRollback {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Error().
					Stack().
					Err(rollbackErr).
					Int64("task.id", taskID).
					Msg("failed to rollback transaction")
			}
		}
	}()

	// Update task status to done within transaction
	task.Status = string(models.TaskStatusDone)
	now := time.Now().UTC()
	task.FinishDate = &now
	err = s.taskRepo.UpdateTaskWithTx(ctx, tx, task)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to update task status to done in transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	// Queue delete_task message
	// If this fails, we'll rollback the transaction
	taskQueueMessage := map[string]interface{}{
		"task": "worker.delete_task",
		"args": []interface{}{task.ID, "telegram"},
	}

	err = s.producer.Publish(ctx, taskQueueMessage)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to queue delete_task message, rolling back transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Transaction will be rolled back in defer
		return nil, errors.Wrap(err, "failed to queue delete_task message")
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to commit transaction")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	// Mark that we've committed, so defer won't rollback
	shouldRollback = false

	// Handle child task logic after transaction commit
	// If this is a child task and parent is not done, create next child task
	if task.ParentID != nil {
		parentTask, err := s.taskRepo.GetTaskByIDWithoutStatusFilter(ctx, *task.ParentID)
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", taskID).
				Int64("parent.id", *task.ParentID).
				Msg("failed to get parent task for child task logic")
			// Don't fail the operation, just log the error
		} else if parentTask.Status != string(models.TaskStatusDone) && parentTask.CronExpression != nil {
			// Parent is not done and has cron expression - create next child task
			log.Debug().
				Int64("task.id", taskID).
				Int64("parent.id", *task.ParentID).
				Str("cron_expression", *parentTask.CronExpression).
				Msg("creating next child task for parent with cron expression")

			// Calculate next execution time from cron expression
			nextTime := cronexpr.MustParse(*parentTask.CronExpression).Next(time.Now().UTC())

			// Create child task
			childTask := &models.Task{
				Title:                  parentTask.Title,
				Description:            parentTask.Description,
				UserID:                 parentTask.UserID,
				MessengerRelatedUserID: parentTask.MessengerRelatedUserID,
				ParentID:               task.ParentID,
				StartDate:              nextTime,
				FinishDate:             parentTask.FinishDate,
				CronExpression:         nil, // Child tasks don't have cron expression
				RequiresConfirmation:   parentTask.RequiresConfirmation,
				Status:                 string(models.TaskStatusScheduled),
			}

			childTaskID, err := s.taskRepo.CreateTask(ctx, childTask)
			if err != nil {
				log.Error().
					Stack().
					Err(err).
					Int64("task.id", taskID).
					Int64("parent.id", *task.ParentID).
					Msg("failed to create next child task")
				// Don't fail the operation, just log the error
			} else {
				log.Debug().
					Int64("task.id", taskID).
					Int64("parent.id", *task.ParentID).
					Int64("child_task.id", childTaskID).
					Time("child_start_date", nextTime).
					Msg("next child task created successfully")
				span.SetAttributes(attribute.Int64("child_task.id", childTaskID))
			}

			if childTask.MessengerRelatedUserID != nil {
				messengerRelatedUser, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *childTask.MessengerRelatedUserID)
				if err != nil {
					log.Error().
						Stack().
						Err(err).
						Int64("task.id", taskID).
						Int64("child_task.id", childTaskID).
						Msg("failed to get messenger related user for child task queue publish")
					// Don't fail, just log
				} else {
					childTaskQueueMessage := map[string]interface{}{
						"task": "worker.schedule_task",
						"args": []interface{}{
							"telegram",
							messengerRelatedUser.ChatID,
							childTaskID,
							childTask.Title,
							childTask.Description,
							childTask.StartDate,
							childTask.CronExpression,
							childTask.RequiresConfirmation,
						},
					}
					err = s.producer.Publish(ctx, childTaskQueueMessage)
					if err != nil {
						log.Error().
							Stack().
							Err(err).
							Int64("task.id", taskID).
							Int64("child_task.id", childTaskID).
							Msg("failed to queue schedule_task for new child task")
						// Don't fail the operation, just log the error
					} else {
						log.Debug().
							Int64("task.id", taskID).
							Int64("child_task.id", childTaskID).
							Msg("schedule_task queued successfully for new child task")
					}
				}
			}

		}
	}

	// If this is a parent task (has cron_expression), sync changes to child tasks and mark them as done
	if task.CronExpression != nil {
		log.Debug().
			Int64("task.id", taskID).
			Msg("parent task marked as done, syncing changes to child tasks and marking them as done")

		// Get all child tasks
		childTasks, err := s.taskRepo.GetChildTasksByParentID(ctx, taskID)
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", taskID).
				Msg("failed to get child tasks for parent")
			// Don't fail the operation, just log the error
		} else if len(childTasks) > 0 {
			// Begin new transaction for child tasks
			childDB := s.taskRepo.GetDB()
			if childDB == nil {
				log.Error().
					Stack().
					Int64("task.id", taskID).
					Msg("database connection not available for child tasks transaction")
				// Don't fail the operation, just log the error
			} else {
				childTx, err := childDB.BeginTxx(ctx, nil)
				if err != nil {
					log.Error().
						Stack().
						Err(err).
						Int64("task.id", taskID).
						Msg("failed to begin transaction for child tasks")
					// Don't fail the operation, just log the error
				} else {
					var childCommitted = false
					defer func() {
						if !childCommitted {
							if rollbackErr := childTx.Rollback(); rollbackErr != nil {
								// Only log error if transaction wasn't already committed/rolled back
								if rollbackErr.Error() != "sql: transaction has already been committed or rolled back" {
									log.Error().
										Stack().
										Err(rollbackErr).
										Int64("task.id", taskID).
										Msg("failed to rollback child tasks transaction")
								}
							}
						}
					}()

					// Sync parent task changes to child tasks and mark them as done
					now := time.Now().UTC()
					hasErrors := false
					for _, childTask := range childTasks {
						// Skip already done or deleted tasks
						if childTask.Status == string(models.TaskStatusDone) {
							continue
						}

						// Sync changes from parent task to child task (for non-done/non-deleted tasks)
						// Sync title if different
						if childTask.Title != task.Title {
							childTask.Title = task.Title
						}

						// Sync description if different
						if childTask.Description != task.Description {
							childTask.Description = task.Description
						}

						// Sync finish_date from parent task
						if task.FinishDate != nil {
							childTask.FinishDate = task.FinishDate
						}

						// Mark child task as done
						childTask.Status = string(models.TaskStatusDone)
						if childTask.FinishDate == nil {
							childTask.FinishDate = &now
						}

						// Update child task in transaction
						err = s.taskRepo.UpdateTaskWithTx(ctx, childTx, childTask)
						if err != nil {
							log.Error().
								Stack().
								Err(err).
								Int64("task.id", taskID).
								Int64("child_task.id", childTask.ID).
								Msg("failed to update child task with parent changes and mark as done")
							hasErrors = true
							break
						}

						// Queue delete_task message for child task
						childTaskQueueMessage := map[string]interface{}{
							"task": "worker.delete_task",
							"args": []interface{}{childTask.ID, "telegram"},
						}

						err = s.producer.Publish(ctx, childTaskQueueMessage)
						if err != nil {
							log.Error().
								Stack().
								Err(err).
								Int64("task.id", taskID).
								Int64("child_task.id", childTask.ID).
								Msg("failed to queue delete_task message for child task, rolling back transaction")
							hasErrors = true
							break
						}
					}

					// Commit transaction if no errors
					if hasErrors {
						// Transaction will be rolled back in defer
						log.Error().
							Int64("task.id", taskID).
							Msg("failed to sync changes and mark all child tasks as done, transaction rolled back")
					} else {
						err = childTx.Commit()
						if err != nil {
							log.Error().
								Stack().
								Err(err).
								Int64("task.id", taskID).
								Msg("failed to commit child tasks transaction")
							// Transaction will be rolled back in defer
						} else {
							childCommitted = true
							log.Debug().
								Int64("task.id", taskID).
								Int("child_tasks.count", len(childTasks)).
								Msg("parent task changes synced to child tasks and all child tasks marked as done successfully")
							span.SetAttributes(attribute.Int("child_tasks.count", len(childTasks)))
						}
					}
				}
			}
		}
	}

	// Record history (outside transaction, as it's not critical if it fails)
	_, historySpan := s.tracer.Start(ctx, "task_service.record_task_marked_done_history",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
			attribute.Int64("user.id", task.UserID),
			attribute.String("status.old", oldStatus),
			attribute.String("status.new", task.Status),
		))
	statusHistory := &models.TaskHistory{
		TaskID:   taskID,
		UserID:   task.UserID,
		Action:   string(models.TaskHistoryActionStatusChanged),
		OldValue: map[string]interface{}{"status": oldStatus},
		NewValue: map[string]interface{}{"status": task.Status},
	}
	if err := s.taskHistoryRepo.CreateTaskHistory(ctx, statusHistory); err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to record task marked as done history")
		historySpan.RecordError(err)
		historySpan.SetStatus(codes.Error, err.Error())
	} else {
		historySpan.SetStatus(codes.Ok, "task marked as done history recorded")
	}
	historySpan.End()

	log.Debug().
		Int64("task.id", taskID).
		Msg("task marked as done successfully")
	span.SetStatus(codes.Ok, "task marked as done successfully")
	return task, nil
}

// GetTaskHistory implements BL of retrieving task history by task ID
func (s *TaskService) GetTaskHistory(ctx context.Context, taskID int64) ([]*models.TaskHistory, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.GetTaskHistory",
		trace.WithAttributes(
			attribute.Int64("task.id", taskID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("task.id", taskID).
		Msg("getting task history")

	// Check if task exists
	_, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", taskID).
			Msg("task not found when getting history")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	histories, err := s.taskHistoryRepo.GetTaskHistoryByTaskID(ctx, taskID)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("task.id", taskID).
			Msg("failed to get task history")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("task.id", taskID).
		Int("history.count", len(histories)).
		Msg("task history retrieved successfully")
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

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int64("user.id", userID).
		Int("limit", limit).
		Int("offset", offset).
		Msg("getting user task history")

	// Check if user exists
	_, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("user not found when getting task history")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	histories, err := s.taskHistoryRepo.GetTaskHistoryByUserID(ctx, userID, limit, offset)
	if err != nil {
		log.Debug().
			Err(err).
			Int64("user.id", userID).
			Msg("failed to get user task history")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	log.Debug().
		Int64("user.id", userID).
		Int("history.count", len(histories)).
		Msg("user task history retrieved successfully")
	span.SetAttributes(attribute.Int("history.count", len(histories)))
	span.SetStatus(codes.Ok, "user task history retrieved successfully")
	return histories, nil
}

// taskToMap converts a task to a map for history storage
func (s *TaskService) taskToMap(task *models.Task) map[string]interface{} {
	result := map[string]interface{}{
		"id":                    task.ID,
		"title":                 task.Title,
		"description":           task.Description,
		"status":                task.Status,
		"requires_confirmation": task.RequiresConfirmation,
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
	if task.ParentID != nil {
		result["parent_id"] = *task.ParentID
	}

	return result
}

// RescheduleTask reschedules a task by updating its start_date to the next day at the same time
// It also adds a daily cron expression and publishes to the queue
// For tasks with a parent, it checks for conflicts with parent's cron schedule and uses parent's next execution time if conflict found
// The status remains "scheduled"
// If queue publishing fails, the task is NOT rescheduled to prevent data loss
func (s *TaskService) RescheduleTask(ctx context.Context, task *models.Task) error {
	ctx, span := s.tracer.Start(ctx, "task_service.RescheduleTask",
		trace.WithAttributes(
			attribute.Int64("task.id", task.ID),
			attribute.Int64("user.id", task.UserID),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Info().
		Int64("task.id", task.ID).
		Int64("user.id", task.UserID).
		Time("old_start_date", task.StartDate).
		Msg("rescheduling task")

	// Calculate next day at the same time
	oldStartDate := task.StartDate

	nowUTC := time.Now().UTC()

	// Calculate initial next day at same time
	newStartDate := oldStartDate.Add(24 * time.Hour)

	// Advance by 24 hours until newStartDate is strictly in the future (handles past dates)
	for !newStartDate.After(nowUTC) {
		newStartDate = newStartDate.Add(24 * time.Hour)
	}

	// If task has parent_id, check if newStartDate conflicts with parent's next execution
	if task.ParentID != nil {
		parentTask, err := s.taskRepo.GetTaskByIDWithoutStatusFilter(ctx, *task.ParentID)
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", task.ID).
				Int64("parent.id", *task.ParentID).
				Msg("failed to get parent task for conflict check")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return errors.WithStack(err)
		}

		if parentTask.CronExpression != nil {
			// Calculate parent's next execution time from nowUTC
			parentNextTime := cronexpr.MustParse(*parentTask.CronExpression).Next(nowUTC)

			// Check if newStartDate falls on the same day as parent's next execution
			// If they are on the same day, use parent's cron time instead
			newStartDateDay := time.Date(newStartDate.Year(), newStartDate.Month(), newStartDate.Day(), 0, 0, 0, 0, newStartDate.Location())
			parentNextTimeDay := time.Date(parentNextTime.Year(), parentNextTime.Month(), parentNextTime.Day(), 0, 0, 0, 0, parentNextTime.Location())

			if newStartDateDay.Equal(parentNextTimeDay) {
				log.Info().
					Int64("task.id", task.ID).
					Int64("parent.id", *task.ParentID).
					Time("new_start_date", newStartDate).
					Time("parent_next_time", parentNextTime).
					Msg("rescheduling aligned to parent cron: conflict detected, using parent execution time")
				span.SetAttributes(
					attribute.String("reason", "conflict_with_parent"),
					attribute.String("parent_next_time", parentNextTime.Format(time.RFC3339)),
				)
				span.SetStatus(codes.Ok, "rescheduling aligned to parent cron schedule")

				newStartDate = parentNextTime
			}
		}
	}

	// Store old status
	oldStatus := task.Status

	// Publish to queue BEFORE updating the task
	// This ensures we don't lose the task if queue publishing fails
	if task.MessengerRelatedUserID == nil {
		err := errors.Wrap(errs.ErrUnprocessableEntity, fmt.Sprintf("task with ID %d has no MessengerRelatedUserID value, cannot reschedule", task.ID))
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", task.ID).
			Msg("cannot reschedule task without messenger related user")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	messengerRelatedUser, err := s.messengerRepo.GetMessengerRelatedUserByID(ctx, *task.MessengerRelatedUserID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			err = errors.Wrap(errs.ErrUnprocessableEntity, err.Error())
		}
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", task.ID).
			Int("messenger_related_user.id", *task.MessengerRelatedUserID).
			Msg("messenger related user not found, cannot reschedule")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	// Prepare task data for queue with new start date and cron expression
	taskQueueMessage := map[string]interface{}{
		"task": "worker.schedule_task",
		"args": []interface{}{"telegram", messengerRelatedUser.ChatID, task.ID, task.Title, task.Description, newStartDate, nil, task.RequiresConfirmation},
	}

	// Publish to queue - if this fails, we don't reschedule
	log.Info().
		Int64("task.id", task.ID).
		Time("new_start_date", newStartDate).
		Msg("publishing rescheduled task to queue")

	err = s.producer.Publish(ctx, taskQueueMessage)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", task.ID).
			Time("new_start_date", newStartDate).
			Msg("failed to publish rescheduled task to queue - task will not be rescheduled to prevent data loss")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Return error so the task is not rescheduled - this prevents data loss
		return errors.Wrap(err, "failed to publish rescheduled task to queue, task not rescheduled")
	}

	log.Info().
		Int64("task.id", task.ID).
		Msg("task published to queue successfully, proceeding with rescheduling")

	// Update the task's start date and cron expression
	task.StartDate = newStartDate
	task.Status = string(models.TaskStatusRescheduled)

	// Update the task in the repository
	err = s.taskRepo.UpdateTask(ctx, task)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", task.ID).
			Int64("user.id", task.UserID).
			Msg("failed to update task after successful queue publishing")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Note: Task was already published to queue, but update failed
		// This is a data inconsistency issue that should be monitored
		return errors.WithStack(err)
	}

	// Record history
	_, historySpan := s.tracer.Start(ctx, "task_service.record_task_rescheduled_history",
		trace.WithAttributes(
			attribute.Int64("task.id", task.ID),
			attribute.Int64("user.id", task.UserID),
		))
	oldValue := map[string]interface{}{"start_date": oldStartDate, "status": oldStatus}
	history := &models.TaskHistory{
		TaskID:   task.ID,
		UserID:   task.UserID,
		Action:   string(models.TaskHistoryActionUpdated),
		OldValue: oldValue,
		NewValue: map[string]interface{}{
			"start_date": newStartDate,
			"status":     task.Status,
		},
	}
	if err := s.taskHistoryRepo.CreateTaskHistory(ctx, history); err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", task.ID).
			Msg("failed to record task rescheduling history")
		historySpan.RecordError(err)
		historySpan.SetStatus(codes.Error, err.Error())
	} else {
		historySpan.SetStatus(codes.Ok, "rescheduling history recorded")
	}
	historySpan.End()

	log.Info().
		Int64("task.id", task.ID).
		Int64("user.id", task.UserID).
		Time("old_start_date", oldStartDate).
		Time("new_start_date", newStartDate).
		Msg("task rescheduled successfully with daily cron expression")
	span.SetAttributes(
		attribute.String("old_start_date", oldStartDate.Format(time.RFC3339)),
		attribute.String("new_start_date", newStartDate.Format(time.RFC3339)),
	)
	span.SetStatus(codes.Ok, "task rescheduled successfully")
	return nil
}

// RescheduleTasks reschedules multiple tasks that have passed their start date
func (s *TaskService) RescheduleTasks(ctx context.Context, tasks []*models.Task) error {
	ctx, span := s.tracer.Start(ctx, "task_service.RescheduleTasks",
		trace.WithAttributes(
			attribute.Int("tasks.count", len(tasks)),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Info().
		Int("tasks.count", len(tasks)).
		Msg("rescheduling tasks")

	var rescheduledCount int
	var failedCount int

	for _, task := range tasks {
		if err := s.RescheduleTask(ctx, task); err != nil {
			failedCount++
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", task.ID).
				Msg("failed to reschedule task")
		} else {
			rescheduledCount++
		}
	}

	log.Info().
		Int("tasks.count", len(tasks)).
		Int("rescheduled.count", rescheduledCount).
		Int("failed.count", failedCount).
		Msg("task rescheduling completed")
	span.SetAttributes(
		attribute.Int("rescheduled.count", rescheduledCount),
		attribute.Int("failed.count", failedCount),
	)
	span.SetStatus(codes.Ok, "tasks rescheduling completed")
	return nil
}

// RescheduleCronTasks updates start_date for tasks with cron expression and requires_confirmation = false
// that have passed their start_date. It calculates the next execution time from cron expression
// and updates only the start_date field without publishing to queue.
func (s *TaskService) RescheduleCronTasks(ctx context.Context, tasks []*models.Task) error {
	ctx, span := s.tracer.Start(ctx, "task_service.RescheduleCronTasks",
		trace.WithAttributes(
			attribute.Int("tasks.count", len(tasks)),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Info().
		Int("tasks.count", len(tasks)).
		Msg("rescheduling cron tasks")

	var updatedCount int
	var failedCount int

	for _, task := range tasks {
		if task.CronExpression == nil {
			log.Warn().
				Int64("task.id", task.ID).
				Msg("task has no cron expression, skipping")
			continue
		}

		// Calculate next execution time from cron expression
		// Use current time as base to get the next occurrence
		// Since we filter by start_date < NOW(), all tasks have passed their start_date
		now := time.Now().UTC()
		nextTime := cronexpr.MustParse(*task.CronExpression).Next(now)

		log.Info().
			Int64("task.id", task.ID).
			Time("old_start_date", task.StartDate).
			Time("new_start_date", nextTime).
			Str("cron_expression", *task.CronExpression).
			Msg("updating start_date for cron task")

		// Update only the start_date field
		task.StartDate = nextTime

		// Update the task in the repository
		err := s.taskRepo.UpdateTask(ctx, task)
		if err != nil {
			failedCount++
			log.Error().
				Stack().
				Err(err).
				Int64("task.id", task.ID).
				Msg("failed to update cron task start_date")
		} else {
			updatedCount++
			log.Info().
				Int64("task.id", task.ID).
				Time("new_start_date", nextTime).
				Msg("cron task start_date updated successfully")
		}
	}

	log.Info().
		Int("tasks.count", len(tasks)).
		Int("updated.count", updatedCount).
		Int("failed.count", failedCount).
		Msg("cron tasks rescheduling completed")
	span.SetAttributes(
		attribute.Int("updated.count", updatedCount),
		attribute.Int("failed.count", failedCount),
	)
	span.SetStatus(codes.Ok, "cron tasks rescheduling completed")
	return nil
}

// GetAllTasks implements BL of retrieving all tasks with pagination, ordering, and filtering
func (s *TaskService) GetAllTasks(ctx context.Context, page, pageSize int, orderBy string, status *string, startDateFrom *time.Time, startDateTo *time.Time, userID *int64) ([]*models.Task, int, error) {
	ctx, span := s.tracer.Start(ctx, "task_service.GetAllTasks",
		trace.WithAttributes(
			attribute.Int("page", page),
			attribute.Int("page_size", pageSize),
			attribute.String("order_by", orderBy),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Debug().
		Int("page", page).
		Int("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting all tasks")

	if status != nil {
		span.SetAttributes(attribute.String("filter.status", *status))
		log = log.With().Str("filter.status", *status).Logger()
	}
	if startDateFrom != nil {
		span.SetAttributes(attribute.String("filter.start_date_from", startDateFrom.Format(time.RFC3339)))
		log = log.With().Time("filter.start_date_from", *startDateFrom).Logger()
	}
	if startDateTo != nil {
		span.SetAttributes(attribute.String("filter.start_date_to", startDateTo.Format(time.RFC3339)))
		log = log.With().Time("filter.start_date_to", *startDateTo).Logger()
	}
	if userID != nil {
		span.SetAttributes(attribute.Int64("filter.user_id", *userID))
		log = log.With().Int64("filter.user_id", *userID).Logger()
	}

	tasks, totalCount, err := s.taskRepo.GetAllTasks(ctx, page, pageSize, orderBy, status, startDateFrom, startDateTo, userID)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to get all tasks")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, errors.WithStack(err)
	}

	log.Debug().
		Int("tasks.count", len(tasks)).
		Int("total_count", totalCount).
		Msg("tasks retrieved successfully")
	span.SetAttributes(
		attribute.Int("tasks.count", len(tasks)),
		attribute.Int("total_count", totalCount),
	)
	span.SetStatus(codes.Ok, "tasks retrieved successfully")
	return tasks, totalCount, nil
}
