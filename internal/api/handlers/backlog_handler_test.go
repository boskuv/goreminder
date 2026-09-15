package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boskuv/goreminder/internal/api/handlers"
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/logger"
)

type stubBacklogService struct {
	createBacklog       func(ctx context.Context, backlog *models.Backlog) (int64, error)
	createBacklogsBatch func(ctx context.Context, items string, separator string, userID int64, messengerRelatedUserID *int) ([]int64, error)
	getBacklogByID      func(ctx context.Context, id int64) (*models.Backlog, error)
	getAllBacklogs      func(ctx context.Context, page, pageSize int, orderBy string, userID *int64, completed *bool, messengerUserID *string) ([]*models.Backlog, int, error)
	updateBacklog       func(ctx context.Context, id int64, updateRequest *models.BacklogUpdateRequest) (*models.Backlog, error)
	deleteBacklog       func(ctx context.Context, id int64) error
}

func (s *stubBacklogService) CreateBacklog(ctx context.Context, backlog *models.Backlog) (int64, error) {
	if s.createBacklog == nil {
		panic("unexpected CreateBacklog")
	}
	return s.createBacklog(ctx, backlog)
}
func (s *stubBacklogService) CreateBacklogsBatch(ctx context.Context, items string, separator string, userID int64, messengerRelatedUserID *int) ([]int64, error) {
	if s.createBacklogsBatch == nil {
		panic("unexpected CreateBacklogsBatch")
	}
	return s.createBacklogsBatch(ctx, items, separator, userID, messengerRelatedUserID)
}
func (s *stubBacklogService) GetBacklogByID(ctx context.Context, id int64) (*models.Backlog, error) {
	if s.getBacklogByID == nil {
		panic("unexpected GetBacklogByID")
	}
	return s.getBacklogByID(ctx, id)
}
func (s *stubBacklogService) GetAllBacklogs(ctx context.Context, page, pageSize int, orderBy string, userID *int64, completed *bool, messengerUserID *string) ([]*models.Backlog, int, error) {
	if s.getAllBacklogs == nil {
		panic("unexpected GetAllBacklogs")
	}
	return s.getAllBacklogs(ctx, page, pageSize, orderBy, userID, completed, messengerUserID)
}
func (s *stubBacklogService) UpdateBacklog(ctx context.Context, id int64, updateRequest *models.BacklogUpdateRequest) (*models.Backlog, error) {
	if s.updateBacklog == nil {
		panic("unexpected UpdateBacklog")
	}
	return s.updateBacklog(ctx, id, updateRequest)
}
func (s *stubBacklogService) DeleteBacklog(ctx context.Context, id int64) error {
	if s.deleteBacklog == nil {
		panic("unexpected DeleteBacklog")
	}
	return s.deleteBacklog(ctx, id)
}

func newBacklogRouter(svc handlers.BacklogService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handlers.NewBacklogHandler(svc, logger.New(io.Discard, zerolog.Disabled, false))
	api := r.Group("/api/v1")
	api.GET("/backlogs", h.GetAllBacklogs)
	api.POST("/backlogs", h.CreateBacklog)
	api.POST("/backlogs/batch", h.CreateBacklogsBatch)
	api.GET("/backlogs/:id", h.GetBacklog)
	api.PUT("/backlogs/:id", h.UpdateBacklog)
	api.DELETE("/backlogs/:id", h.DeleteBacklog)
	return r
}

func TestBacklogHandler_CreateBacklog_Success(t *testing.T) {
	svc := &stubBacklogService{
		createBacklog: func(_ context.Context, backlog *models.Backlog) (int64, error) {
			assert.Equal(t, "Buy milk", backlog.Title)
			assert.Equal(t, int64(7), backlog.UserID)
			return 42, nil
		},
	}
	r := newBacklogRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlogs", bytes.NewBufferString(
		`{"title":"Buy milk","user_id":7}`,
	))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(42), body["id"])
}

func TestBacklogHandler_GetBacklog_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &stubBacklogService{
		getBacklogByID: func(_ context.Context, id int64) (*models.Backlog, error) {
			assert.Equal(t, int64(5), id)
			return &models.Backlog{ID: 5, Title: "Item", UserID: 1, CreatedAt: now, UpdatedAt: now}, nil
		},
	}
	r := newBacklogRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backlogs/5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Item", body["title"])
}

func TestBacklogHandler_GetBacklog_NotFound(t *testing.T) {
	svc := &stubBacklogService{
		getBacklogByID: func(context.Context, int64) (*models.Backlog, error) { return nil, errs.ErrNotFound },
	}
	r := newBacklogRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backlogs/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBacklogHandler_GetAllBacklogs_WithFilters(t *testing.T) {
	svc := &stubBacklogService{
		getAllBacklogs: func(_ context.Context, page, pageSize int, orderBy string, userID *int64, completed *bool, _ *string) ([]*models.Backlog, int, error) {
			assert.Equal(t, 1, page)
			assert.Equal(t, 50, pageSize)
			assert.Equal(t, "created_at DESC", orderBy)
			require.NotNil(t, userID)
			assert.Equal(t, int64(3), *userID)
			require.NotNil(t, completed)
			assert.False(t, *completed)
			return []*models.Backlog{{ID: 1, Title: "A", UserID: 3}}, 1, nil
		},
	}
	r := newBacklogRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backlogs?user_id=3&completed=false", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestBacklogHandler_UpdateBacklog_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &stubBacklogService{
		updateBacklog: func(_ context.Context, id int64, req *models.BacklogUpdateRequest) (*models.Backlog, error) {
			assert.Equal(t, int64(8), id)
			require.NotNil(t, req.Title)
			assert.Equal(t, "Updated", *req.Title)
			return &models.Backlog{ID: 8, Title: "Updated", UserID: 1, CreatedAt: now, UpdatedAt: now}, nil
		},
	}
	r := newBacklogRouter(svc)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/backlogs/8", bytes.NewBufferString(`{"title":"Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestBacklogHandler_DeleteBacklog_Success(t *testing.T) {
	svc := &stubBacklogService{
		deleteBacklog: func(_ context.Context, id int64) error {
			assert.Equal(t, int64(4), id)
			return nil
		},
	}
	r := newBacklogRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/backlogs/4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestBacklogHandler_CreateBacklogsBatch_Success(t *testing.T) {
	svc := &stubBacklogService{
		createBacklogsBatch: func(_ context.Context, items, separator string, userID int64, _ *int) ([]int64, error) {
			assert.Equal(t, "A\nB", items)
			assert.Equal(t, "\n", separator)
			assert.Equal(t, int64(2), userID)
			return []int64{10, 11}, nil
		},
	}
	r := newBacklogRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlogs/batch", bytes.NewBufferString(
		`{"items":"A\nB","separator":"\n","user_id":2}`,
	))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(2), body["count"])
}
