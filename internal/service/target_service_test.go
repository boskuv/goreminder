package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	errs "github.com/boskuv/goreminder/internal/errors"
	mock_repository "github.com/boskuv/goreminder/internal/mocks/repository"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setupTargetService(t *testing.T) (*TargetService, *mock_repository.MockTargetRepository, *mock_repository.MockUserRepository, *mock_repository.MockMessengerRepository) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	targetRepo := mock_repository.NewMockTargetRepository(ctrl)
	userRepo := mock_repository.NewMockUserRepository(ctrl)
	messengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)

	service := NewTargetService(targetRepo, userRepo, messengerRepo, testLogger)
	return service, targetRepo, userRepo, messengerRepo
}

// CreateTarget Tests
func TestTargetService_CreateTarget_Success(t *testing.T) {
	service, targetRepo, userRepo, _ := setupTargetService(t)
	ctx := context.Background()
	target := &models.Target{
		UserID:  1,
		Title:   "Test Target",
		Description: "Test Description",
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	targetRepo.EXPECT().CreateTarget(gomock.Any(), target).Return(int64(42), nil)

	id, err := service.CreateTarget(ctx, target)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestTargetService_CreateTarget_WithMessengerRelatedUser_Success(t *testing.T) {
	service, targetRepo, userRepo, messengerRepo := setupTargetService(t)
	ctx := context.Background()
	messengerUserID := 123
	target := &models.Target{
		UserID:                 1,
		Title:                  "Test Target",
		Description:            "Test Description",
		MessengerRelatedUserID: &messengerUserID,
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), messengerUserID).Return(&models.MessengerRelatedUser{ID: int64(messengerUserID)}, nil)
	targetRepo.EXPECT().CreateTarget(gomock.Any(), target).Return(int64(42), nil)

	id, err := service.CreateTarget(ctx, target)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestTargetService_CreateTarget_UserNotFound(t *testing.T) {
	service, _, userRepo, _ := setupTargetService(t)
	ctx := context.Background()
	target := &models.Target{UserID: 1, Title: "Test Target"}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(nil, errs.ErrNotFound)

	id, err := service.CreateTarget(ctx, target)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

func TestTargetService_CreateTarget_MessengerRelatedUserNotFound(t *testing.T) {
	service, _, userRepo, messengerRepo := setupTargetService(t)
	ctx := context.Background()
	messengerUserID := 123
	target := &models.Target{
		UserID:                 1,
		Title:                  "Test Target",
		MessengerRelatedUserID: &messengerUserID,
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), messengerUserID).Return(nil, errs.ErrNotFound)

	id, err := service.CreateTarget(ctx, target)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

func TestTargetService_CreateTarget_RepositoryError(t *testing.T) {
	service, targetRepo, userRepo, _ := setupTargetService(t)
	ctx := context.Background()
	target := &models.Target{
		UserID:  1,
		Title:   "Test Target",
	}
	expectedErr := errors.New("database error")

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	targetRepo.EXPECT().CreateTarget(gomock.Any(), target).Return(int64(0), expectedErr)

	id, err := service.CreateTarget(ctx, target)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "database error")
}

// GetTargetByID Tests
func TestTargetService_GetTargetByID_Success(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)
	expectedTarget := &models.Target{
		ID:     targetID,
		UserID: 1,
		Title:  "Test Target",
	}

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(expectedTarget, nil)

	target, err := service.GetTargetByID(ctx, targetID)
	assert.NoError(t, err)
	assert.Equal(t, expectedTarget, target)
}

func TestTargetService_GetTargetByID_NotFound(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(nil, errs.ErrNotFound)

	target, err := service.GetTargetByID(ctx, targetID)
	assert.Error(t, err)
	assert.Nil(t, target)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestTargetService_GetTargetByID_RepositoryError(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)
	expectedErr := errors.New("database error")

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(nil, expectedErr)

	target, err := service.GetTargetByID(ctx, targetID)
	assert.Error(t, err)
	assert.Nil(t, target)
	assert.Contains(t, err.Error(), "database error")
}

// GetAllTargets Tests
func TestTargetService_GetAllTargets_Success(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"
	userID := int64(1)
	expectedTargets := []*models.Target{
		{ID: 1, UserID: userID, Title: "Target 1"},
		{ID: 2, UserID: userID, Title: "Target 2"},
	}
	totalCount := 2

	targetRepo.EXPECT().GetAllTargets(gomock.Any(), page, pageSize, orderBy, &userID).Return(expectedTargets, totalCount, nil)

	targets, count, err := service.GetAllTargets(ctx, page, pageSize, orderBy, &userID)
	assert.NoError(t, err)
	assert.Equal(t, expectedTargets, targets)
	assert.Equal(t, totalCount, count)
}

func TestTargetService_GetAllTargets_WithoutUserID(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"
	expectedTargets := []*models.Target{
		{ID: 1, UserID: 1, Title: "Target 1"},
		{ID: 2, UserID: 2, Title: "Target 2"},
	}
	totalCount := 2

	targetRepo.EXPECT().GetAllTargets(gomock.Any(), page, pageSize, orderBy, nil).Return(expectedTargets, totalCount, nil)

	targets, count, err := service.GetAllTargets(ctx, page, pageSize, orderBy, nil)
	assert.NoError(t, err)
	assert.Equal(t, expectedTargets, targets)
	assert.Equal(t, totalCount, count)
}

func TestTargetService_GetAllTargets_EmptyList(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"
	userID := int64(1)
	totalCount := 0

	targetRepo.EXPECT().GetAllTargets(gomock.Any(), page, pageSize, orderBy, &userID).Return([]*models.Target{}, totalCount, nil)

	targets, count, err := service.GetAllTargets(ctx, page, pageSize, orderBy, &userID)
	assert.NoError(t, err)
	assert.Empty(t, targets)
	assert.Equal(t, totalCount, count)
}

func TestTargetService_GetAllTargets_RepositoryError(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	expectedErr := errors.New("database error")

	targetRepo.EXPECT().GetAllTargets(gomock.Any(), 1, 50, "created_at DESC", nil).Return(nil, 0, expectedErr)

	targets, count, err := service.GetAllTargets(ctx, 1, 50, "created_at DESC", nil)
	assert.Error(t, err)
	assert.Nil(t, targets)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "database error")
}

// UpdateTarget Tests
func TestTargetService_UpdateTarget_Success_AllFields(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)
	completedAt := time.Now().UTC()
	originalTarget := &models.Target{
		ID:          targetID,
		UserID:      1,
		Title:       "Original Title",
		Description: "Original Description",
	}
	updateReq := &models.TargetUpdateRequest{
		Title:       ptrString("Updated Title"),
		Description: ptrString("Updated Description"),
		CompletedAt: &completedAt,
	}

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(originalTarget, nil)
	targetRepo.EXPECT().UpdateTarget(gomock.Any(), gomock.Any()).Return(nil)

	updatedTarget, err := service.UpdateTarget(ctx, targetID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedTarget.Title)
	assert.Equal(t, "Updated Description", updatedTarget.Description)
	assert.Equal(t, &completedAt, updatedTarget.CompletedAt)
}

func TestTargetService_UpdateTarget_Success_PartialUpdate(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)
	originalTarget := &models.Target{
		ID:          targetID,
		UserID:      1,
		Title:       "Original Title",
		Description: "Original Description",
	}
	updateReq := &models.TargetUpdateRequest{
		Title: ptrString("Updated Title"),
	}

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(originalTarget, nil)
	targetRepo.EXPECT().UpdateTarget(gomock.Any(), gomock.Any()).Return(nil)

	updatedTarget, err := service.UpdateTarget(ctx, targetID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedTarget.Title)
	assert.Equal(t, "Original Description", updatedTarget.Description) // Unchanged
}

func TestTargetService_UpdateTarget_EmptyTitle(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)
	originalTarget := &models.Target{
		ID:    targetID,
		Title: "Original Title",
	}
	updateReq := &models.TargetUpdateRequest{
		Title: ptrString(""),
	}

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(originalTarget, nil)

	target, err := service.UpdateTarget(ctx, targetID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, target)
	assert.Contains(t, err.Error(), "title cannot be empty")
}

func TestTargetService_UpdateTarget_NotFound(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)
	updateReq := &models.TargetUpdateRequest{
		Title: ptrString("Updated Title"),
	}

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(nil, errs.ErrNotFound)

	target, err := service.UpdateTarget(ctx, targetID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, target)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestTargetService_UpdateTarget_RepositoryError(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)
	originalTarget := &models.Target{
		ID:    targetID,
		Title: "Original Title",
	}
	updateReq := &models.TargetUpdateRequest{
		Title: ptrString("Updated Title"),
	}
	expectedErr := errors.New("database error")

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(originalTarget, nil)
	targetRepo.EXPECT().UpdateTarget(gomock.Any(), gomock.Any()).Return(expectedErr)

	target, err := service.UpdateTarget(ctx, targetID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, target)
	assert.Contains(t, err.Error(), "database error")
}

// DeleteTarget Tests
func TestTargetService_DeleteTarget_Success(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(&models.Target{ID: targetID, UserID: 1}, nil)
	targetRepo.EXPECT().DeleteTarget(gomock.Any(), targetID).Return(nil)

	err := service.DeleteTarget(ctx, targetID)
	assert.NoError(t, err)
}

func TestTargetService_DeleteTarget_NotFound(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(nil, errs.ErrNotFound)

	err := service.DeleteTarget(ctx, targetID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestTargetService_DeleteTarget_RepositoryError(t *testing.T) {
	service, targetRepo, _, _ := setupTargetService(t)
	ctx := context.Background()
	targetID := int64(42)
	expectedErr := errors.New("database error")

	targetRepo.EXPECT().GetTargetByID(gomock.Any(), targetID).Return(&models.Target{ID: targetID, UserID: 1}, nil)
	targetRepo.EXPECT().DeleteTarget(gomock.Any(), targetID).Return(expectedErr)

	err := service.DeleteTarget(ctx, targetID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

