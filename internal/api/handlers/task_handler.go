package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/logger"
)

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	logger      zerolog.Logger
	taskService *service.TaskService
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(taskService *service.TaskService, logger zerolog.Logger) *TaskHandler {
	return &TaskHandler{
		logger:      logger,
		taskService: taskService,
	}
}

// @Summary Create a new task
// @Description Creates a new task and associates it with a user
// @Tags Tasks
// @Accept json
// @Produce json
// @Param task body models.Task true "Task to create"
// @Success 201 {object} map[string]int64
// @Failure 400 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var task models.Task // TODO: separate struct
	if err := c.ShouldBindJSON(&task); err != nil {
		log.Info().
			Err(err).
			Msg("invalid request payload for task creation")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("user.id", task.UserID).
		Str("task.title", task.Title).
		Msg("creating task")

	taskID, err := h.taskService.CreateTask(ctx, &task)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", task.UserID).
			Msg("error while adding new task")

		if errors.Is(err, errs.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Int64("user.id", task.UserID).
		Msg("task created successfully")

	c.JSON(http.StatusCreated, gin.H{"id": taskID})
}

// @Summary Get task by ID
// @Description Retrieves a task by its ID
// @Tags Tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks/{id} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("task_id_param", c.Param("id")).
			Msg("invalid task ID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Msg("getting task")

	task, err := h.taskService.GetTask(ctx, taskID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("error while getting task by its id")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task with id `%d` not found", taskID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Msg("task retrieved successfully")

	c.JSON(http.StatusOK, task)
}

// @Summary Get all user's tasks by userID
// @Description Retrieves all tasks by userID
// @Tags Tasks
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} []models.Task
// @Failure 400 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{user_id}/tasks [get]
func (h *TaskHandler) GetUserTasks(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("user_id_param", c.Param("user_id")).
			Msg("invalid user ID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("user.id", userID).
		Msg("getting user tasks")

	tasks, err := h.taskService.GetUserTasks(ctx, userID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", userID).
			Msg("error while getting tasks by userID parameter")
		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": fmt.Sprintf("user with id `%d` not found", userID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("user.id", userID).
		Int("tasks.count", len(tasks)).
		Msg("user tasks retrieved successfully")

	c.JSON(http.StatusOK, tasks)
}

// @Summary Update a task
// @Description Updates a task by its ID
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param task body models.TaskUpdateRequest true "Task update details"
// @Success 200 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks/{id} [put]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("task_id_param", c.Param("id")).
			Msg("invalid task ID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var taskUpdateRequest models.TaskUpdateRequest
	if err := c.ShouldBindJSON(&taskUpdateRequest); err != nil {
		log.Info().
			Err(err).
			Int64("task.id", taskID).
			Msg("invalid request payload for task update")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Msg("updating task")

	updatedTask, err := h.taskService.UpdateTask(ctx, taskID, &taskUpdateRequest)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("error while updating task")

		if errors.Is(err, errs.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task with id `%d` not found", taskID),
			})
			return
		}
		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Msg("task updated successfully")

	c.JSON(http.StatusOK, updatedTask)
}

// @Summary Soft delete a task
// @Description Marks a task as deleted by its ID (soft delete)
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Success 204 {object} nil
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks/{id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		log.Info().
			Err(errors.Wrap(err, "failed to parse taskID")).
			Str("task_id_param", c.Param("id")).
			Msg("invalid task ID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Msg("deleting task")

	err = h.taskService.DeleteTask(ctx, taskID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("error while soft deleting task")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task with id `%d` not found", taskID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Msg("task deleted successfully")

	c.Status(http.StatusNoContent)
}

// @Summary Send task to queue
// @Description Send task to queue with predefined action
// @Tags Tasks
// @Accept json
// @Produce json
// @Param task body models.ScheduledTask true "Task to enqueue"
// @Success 201 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks/queue [post]
func (h *TaskHandler) QueueTask(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var scheduledTask models.ScheduledTask
	if err := c.ShouldBindJSON(&scheduledTask); err != nil {
		log.Info().
			Err(err).
			Msg("invalid request payload for task queue")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", scheduledTask.TaskID).
		Str("action", scheduledTask.Action).
		Msg("queuing task")

	err := h.taskService.QueueTask(ctx, &scheduledTask)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", scheduledTask.TaskID).
			Msg("error while enqueuing task")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", scheduledTask.TaskID).
		Str("action", scheduledTask.Action).
		Msg("task queued successfully")

	c.JSON(http.StatusCreated, gin.H{"id": scheduledTask.TaskID})
}

// @Summary Mark task as done
// @Description Marks a task as done, updates it in the database, and queues worker.delete_task in a transactional manner. If queueing fails, the database update is rolled back.
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} models.Task "Task marked as done successfully"
// @Failure 400 {object} map[string]string "Invalid task ID parameter"
// @Failure 404 {object} map[string]string "Task not found"
// @Failure 500 {object} map[string]string "Internal server error or transaction failure"
// @Router /api/v1/tasks/{id}/done [post]
func (h *TaskHandler) MarkTaskAsDone(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("task_id_param", c.Param("id")).
			Msg("invalid task ID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Msg("marking task as done")

	task, err := h.taskService.MarkTaskAsDone(ctx, taskID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("error while marking task as done")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task with id `%d` not found", taskID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Msg("task marked as done successfully")

	c.JSON(http.StatusOK, task)
}

// @Summary Get task history by task ID
// @Description Retrieves history entries for a specific task
// @Tags Tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} []models.TaskHistory
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tasks/{id}/history [get]
func (h *TaskHandler) GetTaskHistory(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("task_id_param", c.Param("id")).
			Msg("invalid task ID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Msg("getting task history")

	histories, err := h.taskService.GetTaskHistory(ctx, taskID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", taskID).
			Msg("error while getting task history")

		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("task with id `%d` not found", taskID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Int("history.count", len(histories)).
		Msg("task history retrieved successfully")

	c.JSON(http.StatusOK, histories)
}

// @Summary Get task history by user ID
// @Description Retrieves task history entries for a user with pagination
// @Tags Tasks
// @Produce json
// @Param user_id path int true "User ID"
// @Param limit query int false "Limit (default: 50)" default(50)
// @Param offset query int false "Offset (default: 0)" default(0)
// @Success 200 {object} []models.TaskHistory
// @Failure 400 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{user_id}/tasks/history [get]
func (h *TaskHandler) GetUserTaskHistory(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		log.Info().
			Err(err).
			Str("user_id_param", c.Param("user_id")).
			Msg("invalid user ID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	log.Info().
		Int64("user.id", userID).
		Int("limit", limit).
		Int("offset", offset).
		Msg("getting user task history")

	histories, err := h.taskService.GetUserTaskHistory(ctx, userID, limit, offset)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", userID).
			Msg("error while getting user task history")

		if errors.Is(err, errs.ErrUnprocessableEntity) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": fmt.Sprintf("user with id `%d` not found", userID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Int64("user.id", userID).
		Int("history.count", len(histories)).
		Msg("user task history retrieved successfully")

	c.JSON(http.StatusOK, histories)
}
