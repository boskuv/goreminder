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

func setupBacklogService(t *testing.T) (*BacklogService, *mock_repository.MockBacklogRepository, *mock_repository.MockUserRepository, *mock_repository.MockMessengerRepository) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	backlogRepo := mock_repository.NewMockBacklogRepository(ctrl)
	userRepo := mock_repository.NewMockUserRepository(ctrl)
	messengerRepo := mock_repository.NewMockMessengerRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)

	service := NewBacklogService(backlogRepo, userRepo, messengerRepo, testLogger)
	return service, backlogRepo, userRepo, messengerRepo
}

// CreateBacklog Tests
func TestBacklogService_CreateBacklog_Success(t *testing.T) {
	service, backlogRepo, userRepo, _ := setupBacklogService(t)
	ctx := context.Background()
	backlog := &models.Backlog{
		UserID:      1,
		Title:       "Test Backlog",
		Description: "Test Description",
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	backlogRepo.EXPECT().CreateBacklog(gomock.Any(), backlog).Return(int64(42), nil)

	id, err := service.CreateBacklog(ctx, backlog)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestBacklogService_CreateBacklog_WithMessengerRelatedUser_Success(t *testing.T) {
	service, backlogRepo, userRepo, messengerRepo := setupBacklogService(t)
	ctx := context.Background()
	messengerUserID := 123
	backlog := &models.Backlog{
		UserID:                 1,
		Title:                  "Test Backlog",
		Description:            "Test Description",
		MessengerRelatedUserID: &messengerUserID,
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), messengerUserID).Return(&models.MessengerRelatedUser{ID: int64(messengerUserID)}, nil)
	backlogRepo.EXPECT().CreateBacklog(gomock.Any(), backlog).Return(int64(42), nil)

	id, err := service.CreateBacklog(ctx, backlog)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestBacklogService_CreateBacklog_UserNotFound(t *testing.T) {
	service, _, userRepo, _ := setupBacklogService(t)
	ctx := context.Background()
	backlog := &models.Backlog{UserID: 1, Title: "Test Backlog"}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(nil, errs.ErrNotFound)

	id, err := service.CreateBacklog(ctx, backlog)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

func TestBacklogService_CreateBacklog_UserRepositoryError(t *testing.T) {
	service, _, userRepo, _ := setupBacklogService(t)
	ctx := context.Background()
	backlog := &models.Backlog{UserID: 1, Title: "Test Backlog"}
	expectedErr := errors.New("database error")

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(nil, expectedErr)

	id, err := service.CreateBacklog(ctx, backlog)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "database error")
}

func TestBacklogService_CreateBacklog_MessengerRelatedUserNotFound(t *testing.T) {
	service, _, userRepo, messengerRepo := setupBacklogService(t)
	ctx := context.Background()
	messengerUserID := 123
	backlog := &models.Backlog{
		UserID:                 1,
		Title:                  "Test Backlog",
		MessengerRelatedUserID: &messengerUserID,
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), messengerUserID).Return(nil, errs.ErrNotFound)

	id, err := service.CreateBacklog(ctx, backlog)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

func TestBacklogService_CreateBacklog_BacklogRepositoryError(t *testing.T) {
	service, backlogRepo, userRepo, _ := setupBacklogService(t)
	ctx := context.Background()
	backlog := &models.Backlog{UserID: 1, Title: "Test Backlog"}
	expectedErr := errors.New("backlog creation failed")

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	backlogRepo.EXPECT().CreateBacklog(gomock.Any(), backlog).Return(int64(0), expectedErr)

	id, err := service.CreateBacklog(ctx, backlog)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "backlog creation failed")
}

// CreateBacklogsBatch Tests
func TestBacklogService_CreateBacklogsBatch_Success(t *testing.T) {
	service, backlogRepo, userRepo, _ := setupBacklogService(t)
	ctx := context.Background()
	items := "Item 1\nItem 2\nItem 3"
	userID := int64(1)

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
	backlogRepo.EXPECT().CreateBacklog(gomock.Any(), gomock.Any()).Return(int64(1), nil).Times(3)

	ids, err := service.CreateBacklogsBatch(ctx, items, "\n", userID, nil)
	assert.NoError(t, err)
	assert.Len(t, ids, 3)
	assert.Equal(t, int64(1), ids[0])
}

func TestBacklogService_CreateBacklogsBatch_WithCustomSeparator(t *testing.T) {
	service, backlogRepo, userRepo, _ := setupBacklogService(t)
	ctx := context.Background()
	items := "Item 1|Item 2|Item 3"
	userID := int64(1)

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
	backlogRepo.EXPECT().CreateBacklog(gomock.Any(), gomock.Any()).Return(int64(1), nil).Times(3)

	ids, err := service.CreateBacklogsBatch(ctx, items, "|", userID, nil)
	assert.NoError(t, err)
	assert.Len(t, ids, 3)
}

func TestBacklogService_CreateBacklogsBatch_EmptyItems(t *testing.T) {
	service, _, userRepo, _ := setupBacklogService(t)
	ctx := context.Background()
	items := ""
	userID := int64(1)

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)

	ids, err := service.CreateBacklogsBatch(ctx, items, "\n", userID, nil)
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "no valid items")
}

func TestBacklogService_CreateBacklogsBatch_WithWhitespace(t *testing.T) {
	service, backlogRepo, userRepo, _ := setupBacklogService(t)
	ctx := context.Background()
	items := "Item 1\n\nItem 2\n  \nItem 3"
	userID := int64(1)

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
	backlogRepo.EXPECT().CreateBacklog(gomock.Any(), gomock.Any()).Return(int64(1), nil).Times(3)

	ids, err := service.CreateBacklogsBatch(ctx, items, "\n", userID, nil)
	assert.NoError(t, err)
	assert.Len(t, ids, 3)
}

func TestBacklogService_CreateBacklogsBatch_UserNotFound(t *testing.T) {
	service, _, userRepo, _ := setupBacklogService(t)
	ctx := context.Background()
	items := "Item 1"
	userID := int64(1)

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(nil, errs.ErrNotFound)

	ids, err := service.CreateBacklogsBatch(ctx, items, "\n", userID, nil)
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

// GetBacklogByID Tests
func TestBacklogService_GetBacklogByID_Success(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	expectedBacklog := &models.Backlog{
		ID:          1,
		Title:       "Test Backlog",
		Description: "Test Description",
		UserID:      1,
	}

	backlogRepo.EXPECT().GetBacklogByID(gomock.Any(), int64(1)).Return(expectedBacklog, nil)

	backlog, err := service.GetBacklogByID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, expectedBacklog, backlog)
}

func TestBacklogService_GetBacklogByID_NotFound(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()

	backlogRepo.EXPECT().GetBacklogByID(gomock.Any(), int64(1)).Return(nil, errs.ErrNotFound)

	backlog, err := service.GetBacklogByID(ctx, 1)
	assert.Error(t, err)
	assert.Nil(t, backlog)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestBacklogService_GetBacklogByID_RepositoryError(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	expectedErr := errors.New("database error")

	backlogRepo.EXPECT().GetBacklogByID(gomock.Any(), int64(1)).Return(nil, expectedErr)

	backlog, err := service.GetBacklogByID(ctx, 1)
	assert.Error(t, err)
	assert.Nil(t, backlog)
	assert.Contains(t, err.Error(), "database error")
}

// GetAllBacklogs Tests
func TestBacklogService_GetAllBacklogs_Success(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"
	var completed *bool = nil
	expectedBacklogs := []*models.Backlog{
		{ID: 1, Title: "Backlog 1", UserID: 1},
		{ID: 2, Title: "Backlog 2", UserID: 1},
	}
	totalCount := 2

	backlogRepo.EXPECT().GetAllBacklogs(gomock.Any(), page, pageSize, orderBy, nil, completed).Return(expectedBacklogs, totalCount, nil)

	backlogs, count, err := service.GetAllBacklogs(ctx, page, pageSize, orderBy, nil, completed)
	assert.NoError(t, err)
	assert.Equal(t, expectedBacklogs, backlogs)
	assert.Equal(t, totalCount, count)
}

func TestBacklogService_GetAllBacklogs_WithUserID(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"
	userID := int64(1)
	var completed *bool = nil
	expectedBacklogs := []*models.Backlog{
		{ID: 1, Title: "Backlog 1", UserID: userID},
	}
	totalCount := 1

	backlogRepo.EXPECT().GetAllBacklogs(gomock.Any(), page, pageSize, orderBy, &userID, completed).Return(expectedBacklogs, totalCount, nil)

	backlogs, count, err := service.GetAllBacklogs(ctx, page, pageSize, orderBy, &userID, completed)
	assert.NoError(t, err)
	assert.Equal(t, expectedBacklogs, backlogs)
	assert.Equal(t, totalCount, count)
}

func TestBacklogService_GetAllBacklogs_EmptyList(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"
	var completed *bool = nil
	totalCount := 0

	backlogRepo.EXPECT().GetAllBacklogs(gomock.Any(), page, pageSize, orderBy, nil, completed).Return([]*models.Backlog{}, totalCount, nil)

	backlogs, count, err := service.GetAllBacklogs(ctx, page, pageSize, orderBy, nil, completed)
	assert.NoError(t, err)
	assert.Empty(t, backlogs)
	assert.Equal(t, totalCount, count)
}

func TestBacklogService_GetAllBacklogs_DefaultPagination(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	expectedBacklogs := []*models.Backlog{{ID: 1, Title: "Backlog 1"}}
	var completed *bool = nil
	totalCount := 1

	// Test default page (should be 1)
	backlogRepo.EXPECT().GetAllBacklogs(gomock.Any(), 1, 50, "created_at DESC", nil, completed).Return(expectedBacklogs, totalCount, nil)
	backlogs, count, err := service.GetAllBacklogs(ctx, 0, 0, "", nil, completed)
	assert.NoError(t, err)
	assert.Equal(t, expectedBacklogs, backlogs)
	assert.Equal(t, totalCount, count)
}

func TestBacklogService_GetAllBacklogs_RepositoryError(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	expectedErr := errors.New("database error")
	var completed *bool = nil

	backlogRepo.EXPECT().GetAllBacklogs(gomock.Any(), 1, 50, "created_at DESC", nil, completed).Return(nil, 0, expectedErr)

	backlogs, count, err := service.GetAllBacklogs(ctx, 1, 50, "created_at DESC", nil, completed)
	assert.Error(t, err)
	assert.Nil(t, backlogs)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "database error")
}

// UpdateBacklog Tests
func TestBacklogService_UpdateBacklog_Success_AllFields(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	backlogID := int64(1)
	originalBacklog := &models.Backlog{
		ID:          backlogID,
		Title:       "Old Title",
		Description: "Old Description",
		UserID:      1,
	}
	completedAt := time.Now().UTC()
	updateReq := &models.BacklogUpdateRequest{
		Title:       ptrString("New Title"),
		Description: ptrString("New Description"),
		CompletedAt: &completedAt,
	}

	backlogRepo.EXPECT().GetBacklogByID(gomock.Any(), backlogID).Return(originalBacklog, nil)
	backlogRepo.EXPECT().UpdateBacklog(gomock.Any(), gomock.Any()).Return(nil)

	updatedBacklog, err := service.UpdateBacklog(ctx, backlogID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "New Title", updatedBacklog.Title)
	assert.Equal(t, "New Description", updatedBacklog.Description)
	assert.Equal(t, completedAt, *updatedBacklog.CompletedAt)
}

func TestBacklogService_UpdateBacklog_Success_PartialUpdate(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	backlogID := int64(1)
	originalBacklog := &models.Backlog{
		ID:          backlogID,
		Title:       "Original Title",
		Description: "Original Description",
		UserID:      1,
	}
	updateReq := &models.BacklogUpdateRequest{
		Title: ptrString("Updated Title"),
	}

	backlogRepo.EXPECT().GetBacklogByID(gomock.Any(), backlogID).Return(originalBacklog, nil)
	backlogRepo.EXPECT().UpdateBacklog(gomock.Any(), gomock.Any()).Return(nil)

	updatedBacklog, err := service.UpdateBacklog(ctx, backlogID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedBacklog.Title)
	assert.Equal(t, "Original Description", updatedBacklog.Description)
}

func TestBacklogService_UpdateBacklog_EmptyTitle(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	backlogID := int64(1)
	originalBacklog := &models.Backlog{
		ID:     backlogID,
		Title:  "Original Title",
		UserID: 1,
	}
	updateReq := &models.BacklogUpdateRequest{
		Title: ptrString(""),
	}

	backlogRepo.EXPECT().GetBacklogByID(gomock.Any(), backlogID).Return(originalBacklog, nil)

	backlog, err := service.UpdateBacklog(ctx, backlogID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, backlog)
	assert.Contains(t, err.Error(), "title cannot be empty")
}

func TestBacklogService_UpdateBacklog_BacklogNotFound(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	backlogID := int64(1)
	updateReq := &models.BacklogUpdateRequest{Title: ptrString("New Title")}

	backlogRepo.EXPECT().GetBacklogByID(gomock.Any(), backlogID).Return(nil, errs.ErrNotFound)

	backlog, err := service.UpdateBacklog(ctx, backlogID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, backlog)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestBacklogService_UpdateBacklog_UpdateError(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	backlogID := int64(1)
	originalBacklog := &models.Backlog{ID: backlogID, Title: "Original Title", UserID: 1}
	updateReq := &models.BacklogUpdateRequest{Title: ptrString("New Title")}
	expectedErr := errors.New("update failed")

	backlogRepo.EXPECT().GetBacklogByID(gomock.Any(), backlogID).Return(originalBacklog, nil)
	backlogRepo.EXPECT().UpdateBacklog(gomock.Any(), gomock.Any()).Return(expectedErr)

	backlog, err := service.UpdateBacklog(ctx, backlogID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, backlog)
	assert.Contains(t, err.Error(), "update failed")
}

// DeleteBacklog Tests
func TestBacklogService_DeleteBacklog_Success(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	backlogID := int64(1)

	backlogRepo.EXPECT().DeleteBacklog(gomock.Any(), backlogID).Return(nil)

	err := service.DeleteBacklog(ctx, backlogID)
	assert.NoError(t, err)
}

func TestBacklogService_DeleteBacklog_NotFound(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	backlogID := int64(1)

	backlogRepo.EXPECT().DeleteBacklog(gomock.Any(), backlogID).Return(errs.ErrNotFound)

	err := service.DeleteBacklog(ctx, backlogID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestBacklogService_DeleteBacklog_RepositoryError(t *testing.T) {
	service, backlogRepo, _, _ := setupBacklogService(t)
	ctx := context.Background()
	backlogID := int64(1)
	expectedErr := errors.New("delete failed")

	backlogRepo.EXPECT().DeleteBacklog(gomock.Any(), backlogID).Return(expectedErr)

	err := service.DeleteBacklog(ctx, backlogID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete failed")
}
