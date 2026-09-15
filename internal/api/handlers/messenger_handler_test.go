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

type stubMessengerService struct {
	createMessenger             func(ctx context.Context, messenger *models.Messenger) (int64, error)
	getMessenger                func(ctx context.Context, messengerID int64) (*models.Messenger, error)
	getMessengerIDByName        func(ctx context.Context, messengerName string) (int64, error)
	createMessengerRelatedUser  func(ctx context.Context, messengerRelatedUser *models.MessengerRelatedUser) (int64, error)
	getMessengerRelatedUser     func(ctx context.Context, chatID string, messengerUserID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error)
	getUserID                   func(ctx context.Context, messengerUserID string) (int64, error)
	getAllMessengers            func(ctx context.Context, page, pageSize int, orderBy string) ([]*models.Messenger, int, error)
	getAllMessengerRelatedUsers func(ctx context.Context, page, pageSize int, orderBy string, userID *int64, chatID *string) ([]*models.MessengerRelatedUser, int, error)
}

func (s *stubMessengerService) CreateMessenger(ctx context.Context, messenger *models.Messenger) (int64, error) {
	if s.createMessenger == nil {
		panic("unexpected CreateMessenger")
	}
	return s.createMessenger(ctx, messenger)
}
func (s *stubMessengerService) GetMessenger(ctx context.Context, messengerID int64) (*models.Messenger, error) {
	if s.getMessenger == nil {
		panic("unexpected GetMessenger")
	}
	return s.getMessenger(ctx, messengerID)
}
func (s *stubMessengerService) GetMessengerIDByName(ctx context.Context, messengerName string) (int64, error) {
	if s.getMessengerIDByName == nil {
		panic("unexpected GetMessengerIDByName")
	}
	return s.getMessengerIDByName(ctx, messengerName)
}
func (s *stubMessengerService) CreateMessengerRelatedUser(ctx context.Context, messengerRelatedUser *models.MessengerRelatedUser) (int64, error) {
	if s.createMessengerRelatedUser == nil {
		panic("unexpected CreateMessengerRelatedUser")
	}
	return s.createMessengerRelatedUser(ctx, messengerRelatedUser)
}
func (s *stubMessengerService) GetMessengerRelatedUser(ctx context.Context, chatID string, messengerUserID string, userID *int64, messengerID *int64) (*models.MessengerRelatedUser, error) {
	if s.getMessengerRelatedUser == nil {
		panic("unexpected GetMessengerRelatedUser")
	}
	return s.getMessengerRelatedUser(ctx, chatID, messengerUserID, userID, messengerID)
}
func (s *stubMessengerService) GetUserID(ctx context.Context, messengerUserID string) (int64, error) {
	if s.getUserID == nil {
		panic("unexpected GetUserID")
	}
	return s.getUserID(ctx, messengerUserID)
}
func (s *stubMessengerService) GetAllMessengers(ctx context.Context, page, pageSize int, orderBy string) ([]*models.Messenger, int, error) {
	if s.getAllMessengers == nil {
		panic("unexpected GetAllMessengers")
	}
	return s.getAllMessengers(ctx, page, pageSize, orderBy)
}
func (s *stubMessengerService) GetAllMessengerRelatedUsers(ctx context.Context, page, pageSize int, orderBy string, userID *int64, chatID *string) ([]*models.MessengerRelatedUser, int, error) {
	if s.getAllMessengerRelatedUsers == nil {
		panic("unexpected GetAllMessengerRelatedUsers")
	}
	return s.getAllMessengerRelatedUsers(ctx, page, pageSize, orderBy, userID, chatID)
}

func newMessengerRouter(svc handlers.MessengerService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handlers.NewMessengerHandler(svc, logger.New(io.Discard, zerolog.Disabled, false))
	api := r.Group("/api/v1")
	api.GET("/messengers", h.GetAllMessengers)
	api.POST("/messengers", h.CreateMessenger)
	api.GET("/messengers/:messenger_id", h.GetMessenger)
	api.GET("/messengers/by-name/:messenger_name", h.GetMessengerIDByName)
	api.POST("/messengerRelatedUsers", h.CreateMessengerRelatedUser)
	api.GET("/messengerRelatedUsers", h.GetMessengerRelatedUser)
	api.GET("/messengerRelatedUsers/all", h.GetAllMessengerRelatedUsers)
	api.GET("/messengerRelatedUsers/:messenger_user_id/user", h.GetUserID)
	return r
}

func TestMessengerHandler_CreateMessenger_Success(t *testing.T) {
	svc := &stubMessengerService{
		createMessenger: func(_ context.Context, messenger *models.Messenger) (int64, error) {
			assert.Equal(t, "telegram", messenger.Name)
			return 1, nil
		},
	}
	r := newMessengerRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messengers", bytes.NewBufferString(`{"name":"telegram"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(1), body["id"])
}

func TestMessengerHandler_GetMessenger_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &stubMessengerService{
		getMessenger: func(_ context.Context, id int64) (*models.Messenger, error) {
			assert.Equal(t, int64(1), id)
			return &models.Messenger{ID: 1, Name: "telegram", CreatedAt: now}, nil
		},
	}
	r := newMessengerRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messengers/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "telegram", body["name"])
}

func TestMessengerHandler_GetMessenger_NotFound(t *testing.T) {
	svc := &stubMessengerService{
		getMessenger: func(context.Context, int64) (*models.Messenger, error) { return nil, errs.ErrNotFound },
	}
	r := newMessengerRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messengers/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMessengerHandler_GetMessengerIDByName_Success(t *testing.T) {
	svc := &stubMessengerService{
		getMessengerIDByName: func(_ context.Context, name string) (int64, error) {
			assert.Equal(t, "telegram", name)
			return 1, nil
		},
	}
	r := newMessengerRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messengers/by-name/telegram", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(1), body["id"])
}

func TestMessengerHandler_CreateMessengerRelatedUser_Success(t *testing.T) {
	uid := int64(30)
	mid := int64(1)
	svc := &stubMessengerService{
		createMessengerRelatedUser: func(_ context.Context, mru *models.MessengerRelatedUser) (int64, error) {
			require.NotNil(t, mru.UserID)
			assert.Equal(t, uid, *mru.UserID)
			require.NotNil(t, mru.MessengerID)
			assert.Equal(t, mid, *mru.MessengerID)
			assert.Equal(t, "tg-1", mru.MessengerUserID)
			assert.Equal(t, "chat-1", mru.ChatID)
			return 25, nil
		},
	}
	r := newMessengerRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messengerRelatedUsers", bytes.NewBufferString(
		`{"user_id":30,"messenger_id":1,"messenger_user_id":"tg-1","chat_id":"chat-1"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(25), body["id"])
}

func TestMessengerHandler_GetMessengerRelatedUser_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	uid := int64(30)
	mid := int64(1)
	svc := &stubMessengerService{
		getMessengerRelatedUser: func(_ context.Context, chatID, messengerUserID string, userID, messengerID *int64) (*models.MessengerRelatedUser, error) {
			assert.Equal(t, "chat-1", chatID)
			assert.Equal(t, "tg-1", messengerUserID)
			require.NotNil(t, userID)
			assert.Equal(t, uid, *userID)
			require.NotNil(t, messengerID)
			assert.Equal(t, mid, *messengerID)
			return &models.MessengerRelatedUser{
				ID: 25, UserID: &uid, MessengerID: &mid,
				MessengerUserID: "tg-1", ChatID: "chat-1", CreatedAt: now,
			}, nil
		},
	}
	r := newMessengerRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messengerRelatedUsers?chat_id=chat-1&messenger_user_id=tg-1&user_id=30&messenger_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMessengerHandler_GetUserID_Success(t *testing.T) {
	svc := &stubMessengerService{
		getUserID: func(_ context.Context, messengerUserID string) (int64, error) {
			assert.Equal(t, "tg-1", messengerUserID)
			return 30, nil
		},
	}
	r := newMessengerRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messengerRelatedUsers/tg-1/user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(30), body["user_id"])
}

func TestMessengerHandler_GetAllMessengers_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &stubMessengerService{
		getAllMessengers: func(_ context.Context, page, pageSize int, orderBy string) ([]*models.Messenger, int, error) {
			assert.Equal(t, 1, page)
			assert.Equal(t, 50, pageSize)
			assert.Equal(t, "created_at DESC", orderBy)
			return []*models.Messenger{{ID: 1, Name: "telegram", CreatedAt: now}}, 1, nil
		},
	}
	r := newMessengerRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messengers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
