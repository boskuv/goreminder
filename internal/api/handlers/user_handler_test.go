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

type stubUserService struct {
	createUser             func(ctx context.Context, user *models.User) (int64, error)
	getUser                func(ctx context.Context, userID int64) (*models.User, error)
	updateUser             func(ctx context.Context, userID int64, req *models.UserUpdateRequest) (*models.User, error)
	deleteUser             func(ctx context.Context, userID int64) error
	getAllUsers            func(ctx context.Context, page, pageSize int, orderBy string) ([]*models.User, int, error)
	getRecentUserActivity  func(ctx context.Context, limit int) ([]models.UserActivity, error)
}

func (s *stubUserService) CreateUser(ctx context.Context, user *models.User) (int64, error) {
	if s.createUser == nil {
		panic("unexpected CreateUser")
	}
	return s.createUser(ctx, user)
}
func (s *stubUserService) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	if s.getUser == nil {
		panic("unexpected GetUser")
	}
	return s.getUser(ctx, userID)
}
func (s *stubUserService) UpdateUser(ctx context.Context, userID int64, req *models.UserUpdateRequest) (*models.User, error) {
	if s.updateUser == nil {
		panic("unexpected UpdateUser")
	}
	return s.updateUser(ctx, userID, req)
}
func (s *stubUserService) DeleteUser(ctx context.Context, userID int64) error {
	if s.deleteUser == nil {
		panic("unexpected DeleteUser")
	}
	return s.deleteUser(ctx, userID)
}
func (s *stubUserService) GetAllUsers(ctx context.Context, page, pageSize int, orderBy string) ([]*models.User, int, error) {
	if s.getAllUsers == nil {
		panic("unexpected GetAllUsers")
	}
	return s.getAllUsers(ctx, page, pageSize, orderBy)
}
func (s *stubUserService) GetRecentUserActivity(ctx context.Context, limit int) ([]models.UserActivity, error) {
	if s.getRecentUserActivity == nil {
		panic("unexpected GetRecentUserActivity")
	}
	return s.getRecentUserActivity(ctx, limit)
}

func newUserRouter(svc handlers.UserService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handlers.NewUserHandler(svc, logger.New(io.Discard, zerolog.Disabled, false))
	api := r.Group("/api/v1")
	api.GET("/users", h.GetAllUsers)
	api.POST("/users", h.CreateUser)
	api.GET("/users/activity", h.GetRecentUserActivity)
	api.GET("/users/:user_id", h.GetUser)
	api.PUT("/users/:user_id", h.UpdateUser)
	api.DELETE("/users/:user_id", h.DeleteUser)
	return r
}

func TestUserHandler_GetUser_Success(t *testing.T) {
	svc := &stubUserService{
		getUser: func(_ context.Context, userID int64) (*models.User, error) {
			assert.Equal(t, int64(1), userID)
			return &models.User{ID: 1, Name: "Alice", Email: "a@example.com"}, nil
		},
	}
	r := newUserRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Alice", body["name"])
}

func TestUserHandler_GetUser_NotFound(t *testing.T) {
	svc := &stubUserService{
		getUser: func(context.Context, int64) (*models.User, error) { return nil, errs.ErrNotFound },
	}
	r := newUserRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/9", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserHandler_CreateUser_Success(t *testing.T) {
	svc := &stubUserService{
		createUser: func(_ context.Context, user *models.User) (int64, error) {
			assert.Equal(t, "Bob", user.Name)
			assert.Equal(t, "b@example.com", user.Email)
			return 12, nil
		},
	}
	r := newUserRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(
		`{"name":"Bob","email":"b@example.com","password_hash":"hash"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(12), body["id"])
}

func TestUserHandler_DeleteUser_Success(t *testing.T) {
	svc := &stubUserService{
		deleteUser: func(_ context.Context, userID int64) error {
			assert.Equal(t, int64(3), userID)
			return nil
		},
	}
	r := newUserRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestUserHandler_GetAllUsers_Success(t *testing.T) {
	svc := &stubUserService{
		getAllUsers: func(_ context.Context, page, pageSize int, orderBy string) ([]*models.User, int, error) {
			assert.Equal(t, 1, page)
			assert.Equal(t, 50, pageSize)
			assert.Equal(t, "created_at DESC", orderBy)
			return []*models.User{{ID: 1, Name: "A", Email: "a@x"}}, 1, nil
		},
	}
	r := newUserRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetRecentUserActivity_Success(t *testing.T) {
	at := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	svc := &stubUserService{
		getRecentUserActivity: func(_ context.Context, limit int) ([]models.UserActivity, error) {
			assert.Equal(t, 100, limit)
			return []models.UserActivity{
				{UserID: 1, Name: "Alice", LastActivityAt: at},
			}, nil
		},
	}
	r := newUserRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/activity", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body, 1)
	assert.Equal(t, float64(1), body[0]["user_id"])
	assert.Equal(t, "Alice", body[0]["name"])
	assert.Equal(t, "2026-09-18T12:00:00Z", body[0]["last_activity_at"])
}
