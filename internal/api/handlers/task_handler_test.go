package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/rs/zerolog"

	"github.com/boskuv/goreminder/internal/api/handlers"
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/attachments"
	"github.com/boskuv/goreminder/pkg/logger"
)

// stubTaskService is a minimal TaskService for HTTP wiring tests.
// Set function fields for the endpoints under test; unset methods panic if called.
type stubTaskService struct {
	getTask       func(ctx context.Context, taskID int64) (*models.Task, error)
	muteTask      func(ctx context.Context, taskID int64) (*models.Task, error)
	markTaskAsDone func(ctx context.Context, taskID int64) (*models.Task, error)
}

func (s *stubTaskService) CreateTask(context.Context, *models.Task) (int64, int64, error) {
	panic("unexpected CreateTask")
}
func (s *stubTaskService) GetTask(ctx context.Context, taskID int64) (*models.Task, error) {
	return s.getTask(ctx, taskID)
}
func (s *stubTaskService) GetUserTasks(context.Context, int64, int, int, string, *time.Time, *time.Time, *time.Time, *time.Time, *bool, *string, *string, *string, *bool, *bool, *int, *string) ([]*models.Task, int, error) {
	panic("unexpected GetUserTasks")
}
func (s *stubTaskService) UpdateTask(context.Context, int64, *models.TaskUpdateRequest) (*models.Task, error) {
	panic("unexpected UpdateTask")
}
func (s *stubTaskService) DeleteTask(context.Context, int64) error {
	panic("unexpected DeleteTask")
}
func (s *stubTaskService) QueueTask(context.Context, *models.ScheduledTask) error {
	panic("unexpected QueueTask")
}
func (s *stubTaskService) MarkTaskAsDone(ctx context.Context, taskID int64) (*models.Task, error) {
	return s.markTaskAsDone(ctx, taskID)
}
func (s *stubTaskService) MuteTask(ctx context.Context, taskID int64) (*models.Task, error) {
	return s.muteTask(ctx, taskID)
}
func (s *stubTaskService) UnmuteTask(context.Context, int64) (*models.Task, error) {
	panic("unexpected UnmuteTask")
}
func (s *stubTaskService) GetTaskHistory(context.Context, int64) ([]*models.TaskHistory, error) {
	panic("unexpected GetTaskHistory")
}
func (s *stubTaskService) GetUserTaskHistory(context.Context, int64, int, int) ([]*models.TaskHistory, error) {
	panic("unexpected GetUserTaskHistory")
}
func (s *stubTaskService) GetAllTasks(context.Context, int, int, string, *string, *string, *time.Time, *time.Time, *int64, *string, *bool, *bool, *bool) ([]*models.Task, int, error) {
	panic("unexpected GetAllTasks")
}

func newTaskRouter(svc handlers.TaskService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handlers.NewTaskHandler(svc, attachments.NewNoopClient(), false, logger.New(io.Discard, zerolog.Disabled, false))
	api := r.Group("/api/v1")
	api.GET("/tasks/:id", h.GetTask)
	api.POST("/tasks/:id/mute", h.MuteTask)
	api.POST("/tasks/:id/done", h.MarkTaskAsDone)
	return r
}

func TestTaskHandler_GetTask_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &stubTaskService{
		getTask: func(_ context.Context, taskID int64) (*models.Task, error) {
			assert.Equal(t, int64(42), taskID)
			return &models.Task{
				ID:        42,
				Title:     "hello",
				UserID:    1,
				Status:    string(models.TaskStatusScheduled),
				StartDate: now,
				CreatedAt: now,
			}, nil
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

func TestTaskHandler_MuteTask_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &stubTaskService{
		muteTask: func(_ context.Context, taskID int64) (*models.Task, error) {
			assert.Equal(t, int64(7), taskID)
			return &models.Task{
				ID:        7,
				Title:     "muted",
				UserID:    1,
				Muted:     true,
				Status:    string(models.TaskStatusScheduled),
				StartDate: now,
				CreatedAt: now,
			}, nil
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
