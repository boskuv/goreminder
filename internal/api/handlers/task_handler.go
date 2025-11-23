package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/internal/api/dto/mapper"
	"github.com/boskuv/goreminder/internal/api/validation"
	errs "github.com/boskuv/goreminder/internal/errors"
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
// @Param task body dto.CreateTaskRequest true "Task to create"
// @Success 201 {object} map[string]int64 "Created task ID"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Msg("invalid request payload for task creation")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("user.id", req.UserID).
		Str("task.title", req.Title).
		Msg("creating task")

	// Convert DTO to model for service
	taskModel := mapper.CreateTaskRequestToModel(&req)

	taskID, err := h.taskService.CreateTask(ctx, taskModel)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("user.id", req.UserID).
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
		Int64("user.id", req.UserID).
		Msg("task created successfully")

	c.JSON(http.StatusCreated, gin.H{"id": taskID})
}

// @Summary Get task by ID
// @Description Retrieves a task by its ID
// @Tags Tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} dto.TaskResponse "Task details"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Task not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/tasks/{id} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := validation.ValidateInt64Param(c, "id")
	if err != nil {
		log.Info().
			Err(err).
			Str("task_id_param", c.Param("id")).
			Msg("invalid task ID parameter")
		validation.HandleValidationError(c, err)
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

	// Convert model to response DTO
	response := mapper.TaskModelToResponse(task)
	c.JSON(http.StatusOK, response)
}

// @Summary Get all user's tasks by userID
// @Description Retrieves all tasks by userID
// @Tags Tasks
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {array} dto.TaskResponse "List of tasks"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/users/{user_id}/tasks [get]
func (h *TaskHandler) GetUserTasks(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := validation.ValidateInt64Param(c, "user_id")
	if err != nil {
		log.Info().
			Err(err).
			Str("user_id_param", c.Param("user_id")).
			Msg("invalid user ID parameter")
		validation.HandleValidationError(c, err)
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

	// Convert models to response DTOs
	responses := mapper.TasksModelToResponse(tasks)
	c.JSON(http.StatusOK, responses)
}

// @Summary Update a task
// @Description Updates a task by its ID
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param task body dto.UpdateTaskRequest true "Task update details"
// @Success 200 {object} dto.TaskResponse "Updated task"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Task not found"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/tasks/{id} [put]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := validation.ValidateInt64Param(c, "id")
	if err != nil {
		log.Info().
			Err(err).
			Str("task_id_param", c.Param("id")).
			Msg("invalid task ID parameter")
		validation.HandleValidationError(c, err)
		return
	}

	var req dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Int64("task.id", taskID).
			Msg("invalid request payload for task update")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("task.id", taskID).
		Msg("updating task")

	// Convert DTO to model update request
	updateRequest := mapper.UpdateTaskRequestToModel(&req)

	updatedTask, err := h.taskService.UpdateTask(ctx, taskID, updateRequest)
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

	// Convert model to response DTO
	response := mapper.TaskModelToResponse(updatedTask)
	c.JSON(http.StatusOK, response)
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

	taskID, err := validation.ValidateInt64Param(c, "id")
	if err != nil {
		log.Info().
			Err(err).
			Str("task_id_param", c.Param("id")).
			Msg("invalid task ID parameter")
		validation.HandleValidationError(c, err)
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
// @Param task body dto.QueueTaskRequest true "Task to enqueue"
// @Success 201 {object} map[string]int64 "Task ID"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Task not found"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/tasks/queue [post]
func (h *TaskHandler) QueueTask(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	var req dto.QueueTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info().
			Err(err).
			Msg("invalid request payload for task queue")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("task.id", req.TaskID).
		Str("action", req.Action).
		Msg("queuing task")

	// Convert DTO to model for service
	scheduledModel := mapper.QueueTaskRequestToModel(&req)

	err := h.taskService.QueueTask(ctx, scheduledModel)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Int64("task.id", req.TaskID).
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
		Int64("task.id", req.TaskID).
		Str("action", req.Action).
		Msg("task queued successfully")

	c.JSON(http.StatusCreated, gin.H{"id": req.TaskID})
}

// @Summary Mark task as done
// @Description Marks a task as done, updates it in the database, and queues worker.delete_task in a transactional manner. If queueing fails, the database update is rolled back.
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} dto.TaskResponse "Task marked as done successfully"
// @Failure 400 {object} map[string]string "Invalid task ID parameter"
// @Failure 404 {object} map[string]string "Task not found"
// @Failure 500 {object} map[string]string "Internal server error or transaction failure"
// @Router /api/v1/tasks/{id}/done [post]
func (h *TaskHandler) MarkTaskAsDone(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := validation.ValidateInt64Param(c, "id")
	if err != nil {
		log.Info().
			Err(err).
			Str("task_id_param", c.Param("id")).
			Msg("invalid task ID parameter")
		validation.HandleValidationError(c, err)
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

	// Convert model to response DTO
	response := mapper.TaskModelToResponse(task)
	c.JSON(http.StatusOK, response)
}

// @Summary Get task history by task ID
// @Description Retrieves history entries for a specific task
// @Tags Tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {array} dto.TaskHistoryResponse "Task history entries"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Task not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/tasks/{id}/history [get]
func (h *TaskHandler) GetTaskHistory(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	taskID, err := validation.ValidateInt64Param(c, "id")
	if err != nil {
		log.Info().
			Err(err).
			Str("task_id_param", c.Param("id")).
			Msg("invalid task ID parameter")
		validation.HandleValidationError(c, err)
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

	// Convert models to response DTOs
	responses := mapper.TaskHistoriesModelToResponse(histories)
	c.JSON(http.StatusOK, responses)
}

// @Summary Get task history by user ID
// @Description Retrieves task history entries for a user with pagination
// @Tags Tasks
// @Produce json
// @Param user_id path int true "User ID"
// @Param limit query int false "Limit (default: 50)" default(50)
// @Param offset query int false "Offset (default: 0)" default(0)
// @Success 200 {array} dto.TaskHistoryResponse "Task history entries"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 422 {object} map[string]string "Unprocessable entity"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/users/{user_id}/tasks/history [get]
func (h *TaskHandler) GetUserTaskHistory(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	userID, err := validation.ValidateInt64Param(c, "user_id")
	if err != nil {
		log.Info().
			Err(err).
			Str("user_id_param", c.Param("user_id")).
			Msg("invalid user ID parameter")
		validation.HandleValidationError(c, err)
		return
	}

	limit, err := validation.ValidateInt64Query(c, "limit", 50, 1)
	if err != nil {
		log.Info().
			Err(err).
			Msg("invalid limit query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	offset, err := validation.ValidateInt64Query(c, "offset", 0, 0)
	if err != nil {
		log.Info().
			Err(err).
			Msg("invalid offset query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("user.id", userID).
		Int64("limit", limit).
		Int64("offset", offset).
		Msg("getting user task history")

	histories, err := h.taskService.GetUserTaskHistory(ctx, userID, int(limit), int(offset))
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

// @Summary Get all tasks
// @Description Retrieves all tasks with pagination, ordering, and filtering (by status, start_date_from, start_date_to, user_id)
// @Tags Tasks
// @Produce json
// @Param page query int false "Page number (default: 1)" default(1)
// @Param page_size query int false "Page size (default: 50)" default(50)
// @Param order_by query string false "Order by field (default: created_at DESC)" default(created_at DESC)
// @Param status query string false "Filter by status (pending, scheduled, done, rescheduled, postponed, deleted)"
// @Param start_date_from query string false "Filter by start_date from (RFC3339 format, inclusive)"
// @Param start_date_to query string false "Filter by start_date to (RFC3339 format, inclusive)"
// @Param user_id query int false "Filter by user_id"
// @Success 200 {object} dto.PaginatedTasksResponse "Paginated list of tasks"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/tasks [get]
func (h *TaskHandler) GetAllTasks(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.WithTraceContext(ctx, h.logger)

	page, err := validation.ValidateInt64Query(c, "page", 1, 1)
	if err != nil {
		log.Info().Err(err).Msg("invalid page query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	pageSize, err := validation.ValidateInt64Query(c, "page_size", 50, 1)
	if err != nil {
		log.Info().Err(err).Msg("invalid page_size query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	orderBy, err := validation.ValidateOptionalStringQuery(c, "order_by")
	if err != nil {
		log.Info().Err(err).Msg("invalid order_by query parameter")
		validation.HandleValidationError(c, err)
		return
	}
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	// Optional filters
	status, err := validation.ValidateOptionalStringQuery(c, "status")
	if err != nil {
		log.Info().Err(err).Msg("invalid status query parameter")
		validation.HandleValidationError(c, err)
		return
	}
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	startDateFrom, err := validation.ValidateOptionalTimeQuery(c, "start_date_from")
	if err != nil {
		log.Info().Err(err).Msg("invalid start_date_from query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	startDateTo, err := validation.ValidateOptionalTimeQuery(c, "start_date_to")
	if err != nil {
		log.Info().Err(err).Msg("invalid start_date_to query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	userID, err := validation.ValidateOptionalInt64Query(c, "user_id", 1)
	if err != nil {
		log.Info().Err(err).Msg("invalid user_id query parameter")
		validation.HandleValidationError(c, err)
		return
	}

	log.Info().
		Int64("page", page).
		Int64("page_size", pageSize).
		Str("order_by", orderBy).
		Msg("getting all tasks")

	tasks, totalCount, err := h.taskService.GetAllTasks(ctx, int(page), int(pageSize), orderBy, statusPtr, startDateFrom, startDateTo, userID)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Msg("error while getting all tasks")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total pages
	totalPages := (totalCount + int(pageSize) - 1) / int(pageSize)
	if totalPages == 0 {
		totalPages = 1
	}

	// Convert models to response DTOs
	responsesPtr := mapper.TasksModelToResponse(tasks)
	responses := make([]dto.TaskResponse, len(responsesPtr))
	for i, resp := range responsesPtr {
		responses[i] = *resp
	}

	response := dto.PaginatedTasksResponse{
		Data: responses,
		Pagination: dto.PaginationResponse{
			Page:       int(page),
			PageSize:   int(pageSize),
			TotalPages: totalPages,
			TotalCount: totalCount,
		},
	}

	log.Info().
		Int("tasks.count", len(tasks)).
		Int("total_count", totalCount).
		Msg("tasks retrieved successfully")

	c.JSON(http.StatusOK, response)
}
