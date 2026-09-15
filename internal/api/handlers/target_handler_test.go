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

type stubTargetService struct {
	createTarget  func(ctx context.Context, target *models.Target) (int64, error)
	getTargetByID func(ctx context.Context, id int64) (*models.Target, error)
	getAllTargets func(ctx context.Context, page, pageSize int, orderBy string, userID *int64, messengerUserID *string) ([]*models.Target, int, error)
	updateTarget  func(ctx context.Context, id int64, updateRequest *models.TargetUpdateRequest) (*models.Target, error)
	deleteTarget  func(ctx context.Context, id int64) error
}

func (s *stubTargetService) CreateTarget(ctx context.Context, target *models.Target) (int64, error) {
	if s.createTarget == nil {
		panic("unexpected CreateTarget")
	}
	return s.createTarget(ctx, target)
}
func (s *stubTargetService) GetTargetByID(ctx context.Context, id int64) (*models.Target, error) {
	if s.getTargetByID == nil {
		panic("unexpected GetTargetByID")
	}
	return s.getTargetByID(ctx, id)
}
func (s *stubTargetService) GetAllTargets(ctx context.Context, page, pageSize int, orderBy string, userID *int64, messengerUserID *string) ([]*models.Target, int, error) {
	if s.getAllTargets == nil {
		panic("unexpected GetAllTargets")
	}
	return s.getAllTargets(ctx, page, pageSize, orderBy, userID, messengerUserID)
}
func (s *stubTargetService) UpdateTarget(ctx context.Context, id int64, updateRequest *models.TargetUpdateRequest) (*models.Target, error) {
	if s.updateTarget == nil {
		panic("unexpected UpdateTarget")
	}
	return s.updateTarget(ctx, id, updateRequest)
}
func (s *stubTargetService) DeleteTarget(ctx context.Context, id int64) error {
	if s.deleteTarget == nil {
		panic("unexpected DeleteTarget")
	}
	return s.deleteTarget(ctx, id)
}

func newTargetRouter(svc handlers.TargetService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handlers.NewTargetHandler(svc, logger.New(io.Discard, zerolog.Disabled, false))
	api := r.Group("/api/v1")
	api.GET("/targets", h.GetAllTargets)
	api.POST("/targets", h.CreateTarget)
	api.GET("/targets/:id", h.GetTarget)
	api.PUT("/targets/:id", h.UpdateTarget)
	api.DELETE("/targets/:id", h.DeleteTarget)
	return r
}

func TestTargetHandler_CreateTarget_Success(t *testing.T) {
	svc := &stubTargetService{
		createTarget: func(_ context.Context, target *models.Target) (int64, error) {
			assert.Equal(t, "Learn Go", target.Title)
			assert.Equal(t, int64(9), target.UserID)
			return 15, nil
		},
	}
	r := newTargetRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/targets", bytes.NewBufferString(
		`{"title":"Learn Go","user_id":9}`,
	))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(15), body["id"])
}

func TestTargetHandler_GetTarget_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &stubTargetService{
		getTargetByID: func(_ context.Context, id int64) (*models.Target, error) {
			assert.Equal(t, int64(3), id)
			return &models.Target{ID: 3, Title: "Goal", UserID: 1, CreatedAt: now, UpdatedAt: now}, nil
		},
	}
	r := newTargetRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/targets/3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Goal", body["title"])
}

func TestTargetHandler_GetTarget_NotFound(t *testing.T) {
	svc := &stubTargetService{
		getTargetByID: func(context.Context, int64) (*models.Target, error) { return nil, errs.ErrNotFound },
	}
	r := newTargetRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/targets/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTargetHandler_GetAllTargets_WithUserFilter(t *testing.T) {
	svc := &stubTargetService{
		getAllTargets: func(_ context.Context, page, pageSize int, orderBy string, userID *int64, _ *string) ([]*models.Target, int, error) {
			assert.Equal(t, 1, page)
			assert.Equal(t, 50, pageSize)
			assert.Equal(t, "created_at DESC", orderBy)
			require.NotNil(t, userID)
			assert.Equal(t, int64(4), *userID)
			return []*models.Target{{ID: 1, Title: "T", UserID: 4}}, 1, nil
		},
	}
	r := newTargetRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/targets?user_id=4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestTargetHandler_UpdateTarget_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &stubTargetService{
		updateTarget: func(_ context.Context, id int64, req *models.TargetUpdateRequest) (*models.Target, error) {
			assert.Equal(t, int64(6), id)
			require.NotNil(t, req.Title)
			assert.Equal(t, "New", *req.Title)
			return &models.Target{ID: 6, Title: "New", UserID: 1, CreatedAt: now, UpdatedAt: now}, nil
		},
	}
	r := newTargetRouter(svc)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/targets/6", bytes.NewBufferString(`{"title":"New"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestTargetHandler_DeleteTarget_Success(t *testing.T) {
	svc := &stubTargetService{
		deleteTarget: func(_ context.Context, id int64) error {
			assert.Equal(t, int64(2), id)
			return nil
		},
	}
	r := newTargetRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/targets/2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
