package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boskuv/goreminder/internal/api/handlers"
	"github.com/boskuv/goreminder/internal/api/validation"
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/attachments"
	"github.com/boskuv/goreminder/pkg/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = validation.RegisterCustomValidators(v)
	}
}

// stubTaskService is a minimal TaskService for HTTP wiring tests.
type stubTaskService struct {
	createTask     func(ctx context.Context, task *models.Task) (int64, int64, error)
	getTask        func(ctx context.Context, taskID int64) (*models.Task, error)
	updateTask     func(ctx context.Context, taskID int64, req *models.TaskUpdateRequest) (*models.Task, error)
	deleteTask     func(ctx context.Context, taskID int64) error
	muteTask       func(ctx context.Context, taskID int64) (*models.Task, error)
	unmuteTask     func(ctx context.Context, taskID int64) (*models.Task, error)
	markTaskAsDone func(ctx context.Context, taskID int64) (*models.Task, error)
	getAllTasks    func(ctx context.Context, page, pageSize int, orderBy string, status *string, statusNot *string, startDateFrom *time.Time, startDateTo *time.Time, userID *int64, cronExpression *string, cronExpressionIsNull *bool, requiresConfirmation *bool, excludeCronWithConfirmation *bool) ([]*models.Task, int, error)
}

func (s *stubTaskService) CreateTask(ctx context.Context, task *models.Task) (int64, int64, error) {
	if s.createTask == nil {
		panic("unexpected CreateTask")
	}
	return s.createTask(ctx, task)
}
func (s *stubTaskService) GetTask(ctx context.Context, taskID int64) (*models.Task, error) {
	if s.getTask == nil {
		panic("unexpected GetTask")
	}
	return s.getTask(ctx, taskID)
}
func (s *stubTaskService) GetUserTasks(context.Context, int64, int, int, string, *time.Time, *time.Time, *time.Time, *time.Time, *bool, *string, *string, *string, *bool, *bool, *int, *string, *string) ([]*models.Task, int, error) {
	panic("unexpected GetUserTasks")
}
func (s *stubTaskService) UpdateTask(ctx context.Context, taskID int64, req *models.TaskUpdateRequest) (*models.Task, error) {
	if s.updateTask == nil {
		panic("unexpected UpdateTask")
	}
	return s.updateTask(ctx, taskID, req)
}
func (s *stubTaskService) DeleteTask(ctx context.Context, taskID int64) error {
	if s.deleteTask == nil {
		panic("unexpected DeleteTask")
	}
	return s.deleteTask(ctx, taskID)
}
func (s *stubTaskService) QueueTask(context.Context, *models.ScheduledTask) error {
	panic("unexpected QueueTask")
}
func (s *stubTaskService) MarkTaskAsDone(ctx context.Context, taskID int64) (*models.Task, error) {
	if s.markTaskAsDone == nil {
		panic("unexpected MarkTaskAsDone")
	}
	return s.markTaskAsDone(ctx, taskID)
}
func (s *stubTaskService) MuteTask(ctx context.Context, taskID int64) (*models.Task, error) {
	if s.muteTask == nil {
		panic("unexpected MuteTask")
	}
	return s.muteTask(ctx, taskID)
}
func (s *stubTaskService) UnmuteTask(ctx context.Context, taskID int64) (*models.Task, error) {
	if s.unmuteTask == nil {
		panic("unexpected UnmuteTask")
	}
	return s.unmuteTask(ctx, taskID)
}
func (s *stubTaskService) GetTaskHistory(context.Context, int64) ([]*models.TaskHistory, error) {
	panic("unexpected GetTaskHistory")
}
func (s *stubTaskService) GetUserTaskHistory(context.Context, int64, int, int) ([]*models.TaskHistory, error) {
	panic("unexpected GetUserTaskHistory")
}
func (s *stubTaskService) GetAllTasks(ctx context.Context, page, pageSize int, orderBy string, status *string, statusNot *string, startDateFrom *time.Time, startDateTo *time.Time, userID *int64, cronExpression *string, cronExpressionIsNull *bool, requiresConfirmation *bool, excludeCronWithConfirmation *bool, externalProvider *string) ([]*models.Task, int, error) {
	if s.getAllTasks == nil {
		panic("unexpected GetAllTasks")
	}
	return s.getAllTasks(ctx, page, pageSize, orderBy, status, statusNot, startDateFrom, startDateTo, userID, cronExpression, cronExpressionIsNull, requiresConfirmation, excludeCronWithConfirmation)
}

func sampleTask(id int64, title string) *models.Task {
	now := time.Now().UTC().Truncate(time.Second)
	return &models.Task{
		ID:        id,
		Title:     title,
		UserID:    1,
		Status:    string(models.TaskStatusScheduled),
		StartDate: now.Add(time.Hour),
		CreatedAt: now,
	}
}

func newTaskRouter(svc handlers.TaskService) *gin.Engine {
	r := gin.New()
	h := handlers.NewTaskHandler(svc, attachments.NewNoopClient(), false, logger.New(io.Discard, zerolog.Disabled, false))
	api := r.Group("/api/v1")
	api.GET("/tasks", h.GetAllTasks)
	api.POST("/tasks", h.CreateTask)
	api.GET("/tasks/:id", h.GetTask)
	api.PUT("/tasks/:id", h.UpdateTask)
	api.DELETE("/tasks/:id", h.DeleteTask)
	api.POST("/tasks/:id/mute", h.MuteTask)
	api.POST("/tasks/:id/unmute", h.UnmuteTask)
	api.POST("/tasks/:id/done", h.MarkTaskAsDone)
	return r
}

func TestTaskHandler_GetTask_Success(t *testing.T) {
	svc := &stubTaskService{
		getTask: func(_ context.Context, taskID int64) (*models.Task, error) {
			assert.Equal(t, int64(42), taskID)
			return sampleTask(42, "hello"), nil
		},
	}
	r := newTaskRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(42), body["id"])
	assert.Equal(t, "hello", body["title"])
}

func TestTaskHandler_GetTask_NotFound(t *testing.T) {
	svc := &stubTaskService{
		getTask: func(context.Context, int64) (*models.Task, error) {
			return nil, errs.ErrNotFound
		},
	}
	r := newTaskRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTaskHandler_GetTask_InvalidID(t *testing.T) {
	svc := &stubTaskService{
		getTask: func(context.Context, int64) (*models.Task, error) {
			t.Fatal("service must not be called for invalid id")
			return nil, nil
		},
	}
	r := newTaskRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTaskHandler_CreateTask_Success(t *testing.T) {
	start := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	svc := &stubTaskService{
		createTask: func(_ context.Context, task *models.Task) (int64, int64, error) {
			assert.Equal(t, "new task", task.Title)
			assert.Equal(t, int64(1), task.UserID)
			return 55, 0, nil
		},
	}
	r := newTaskRouter(svc)

	payload := fmt.Sprintf(`{"title":"new task","user_id":1,"start_date":"%s"}`, start.Format(time.RFC3339))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(55), body["id"])
}

func TestTaskHandler_CreateTask_ValidationError(t *testing.T) {
	svc := &stubTaskService{
		createTask: func(context.Context, *models.Task) (int64, int64, error) {
			t.Fatal("service must not be called")
			return 0, 0, nil
		},
	}
	r := newTaskRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(`{"title":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTaskHandler_UpdateTask_Success(t *testing.T) {
	svc := &stubTaskService{
		updateTask: func(_ context.Context, taskID int64, req *models.TaskUpdateRequest) (*models.Task, error) {
			assert.Equal(t, int64(3), taskID)
			require.NotNil(t, req.Title)
			assert.Equal(t, "updated", *req.Title)
			task := sampleTask(3, "updated")
			return task, nil
		},
	}
	r := newTaskRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/3", bytes.NewBufferString(`{"title":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "updated", body["title"])
}

func TestTaskHandler_UpdateTask_NotFound(t *testing.T) {
	svc := &stubTaskService{
		updateTask: func(context.Context, int64, *models.TaskUpdateRequest) (*models.Task, error) {
			return nil, errs.ErrNotFound
		},
	}
	r := newTaskRouter(svc)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/3", bytes.NewBufferString(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTaskHandler_DeleteTask_Success(t *testing.T) {
	svc := &stubTaskService{
		deleteTask: func(_ context.Context, taskID int64) error {
			assert.Equal(t, int64(9), taskID)
			return nil
		},
	}
	r := newTaskRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/9", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestTaskHandler_DeleteTask_NotFound(t *testing.T) {
	svc := &stubTaskService{
		deleteTask: func(context.Context, int64) error { return errs.ErrNotFound },
	}
	r := newTaskRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/9", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTaskHandler_MuteTask_Success(t *testing.T) {
	svc := &stubTaskService{
		muteTask: func(_ context.Context, taskID int64) (*models.Task, error) {
			assert.Equal(t, int64(7), taskID)
			task := sampleTask(7, "muted")
			task.Muted = true
			return task, nil
		},
	}
	r := newTaskRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/7/mute", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["muted"])
}

func TestTaskHandler_UnmuteTask_Success(t *testing.T) {
	svc := &stubTaskService{
		unmuteTask: func(_ context.Context, taskID int64) (*models.Task, error) {
			assert.Equal(t, int64(8), taskID)
			task := sampleTask(8, "unmuted")
			task.Muted = false
			return task, nil
		},
	}
	r := newTaskRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/8/unmute", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, false, body["muted"])
}

func TestTaskHandler_MarkTaskAsDone_NotFound(t *testing.T) {
	svc := &stubTaskService{
		markTaskAsDone: func(context.Context, int64) (*models.Task, error) {
			return nil, errs.ErrNotFound
		},
	}
	r := newTaskRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/5/done", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTaskHandler_GetAllTasks_PassesFilters(t *testing.T) {
	svc := &stubTaskService{
		getAllTasks: func(_ context.Context, page, pageSize int, orderBy string, status *string, _ *string, _ *time.Time, _ *time.Time, userID *int64, _ *string, _ *bool, _ *bool, _ *bool) ([]*models.Task, int, error) {
			assert.Equal(t, 2, page)
			assert.Equal(t, 10, pageSize)
			assert.Equal(t, "id DESC", orderBy)
			require.NotNil(t, status)
			assert.Equal(t, "scheduled", *status)
			require.NotNil(t, userID)
			assert.Equal(t, int64(1), *userID)
			return []*models.Task{sampleTask(11, "listed")}, 1, nil
		},
	}
	r := newTaskRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?page=2&page_size=10&order_by=id%20DESC&status=scheduled&user_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body, "data")
	assert.Contains(t, body, "pagination")
}
