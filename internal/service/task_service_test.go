package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	errs "github.com/boskuv/goreminder/internal/errors"
	mock_repositories "github.com/boskuv/goreminder/internal/mocks/repository"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/boskuv/goreminder/pkg/queue"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setup(t *testing.T) (*TaskService, *mock_repositories.MockTaskRepository, *mock_repositories.MockUserRepository, *mock_repositories.MockMessengerRepository, *mock_repositories.MockTaskHistoryRepository, *queue.Producer) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	taskRepo := mock_repositories.NewMockTaskRepository(ctrl)
	userRepo := mock_repositories.NewMockUserRepository(ctrl)
	messengerRepo := mock_repositories.NewMockMessengerRepository(ctrl)
	taskHistoryRepo := mock_repositories.NewMockTaskHistoryRepository(ctrl)
	producer := &queue.Producer{}
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)

	service := NewTaskService(taskRepo, userRepo, messengerRepo, taskHistoryRepo, producer, testLogger)
	return service, taskRepo, userRepo, messengerRepo, taskHistoryRepo, producer
}

// Helper functions
func ptrString(s string) *string     { return &s }
func ptrTime(t time.Time) *time.Time { return &t }
func ptrInt(i int) *int              { return &i }

// CreateTask Tests
func TestTaskService_CreateTask_Success(t *testing.T) {
	service, taskRepo, userRepo, _, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	task := &models.Task{
		UserID:               1,
		Title:                "Test Task",
		Description:          "Test Description",
		Status:               "pending",
		RequiresConfirmation: false,
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	taskRepo.EXPECT().CreateTask(gomock.Any(), task).Return(int64(42), nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	id, childTaskID, err := service.CreateTask(ctx, task)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
	assert.Equal(t, int64(0), childTaskID)
}

func TestTaskService_CreateTask_WithMessengerRelatedUser_Success(t *testing.T) {
	service, taskRepo, userRepo, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	messengerUserID := 123
	messengerID := int64(1)
	task := &models.Task{
		UserID:                 1,
		Title:                  "Test Task",
		Description:            "Test Description",
		Status:                 "pending",
		MessengerRelatedUserID: &messengerUserID,
		RequiresConfirmation:   false,
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), messengerUserID).Return(&models.MessengerRelatedUser{ID: int64(messengerUserID), MessengerID: &messengerID}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "Telegram"}, nil)
	taskRepo.EXPECT().CreateTask(gomock.Any(), task).Return(int64(42), nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	id, childTaskID, err := service.CreateTask(ctx, task)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
	assert.Equal(t, int64(0), childTaskID)
}

func TestTaskService_CreateTask_UserNotFound(t *testing.T) {
	service, _, userRepo, _, _, _ := setup(t)
	ctx := context.Background()
	task := &models.Task{UserID: 1, RequiresConfirmation: false}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(nil, errs.ErrNotFound)

	id, childTaskID, err := service.CreateTask(ctx, task)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Equal(t, int64(0), childTaskID)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

func TestTaskService_CreateTask_UserRepositoryError(t *testing.T) {
	service, _, userRepo, _, _, _ := setup(t)
	ctx := context.Background()
	task := &models.Task{UserID: 1, RequiresConfirmation: false}
	expectedErr := errors.New("database error")

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(nil, expectedErr)

	id, childTaskID, err := service.CreateTask(ctx, task)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Equal(t, int64(0), childTaskID)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_CreateTask_MessengerRelatedUserNotFound(t *testing.T) {
	service, _, userRepo, messengerRepo, _, _ := setup(t)
	ctx := context.Background()
	messengerUserID := 123
	task := &models.Task{
		UserID:                 1,
		MessengerRelatedUserID: &messengerUserID,
		RequiresConfirmation:   false,
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), messengerUserID).Return(nil, errs.ErrNotFound)

	id, childTaskID, err := service.CreateTask(ctx, task)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Equal(t, int64(0), childTaskID)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

func TestTaskService_CreateTask_MessengerRepositoryError(t *testing.T) {
	service, _, userRepo, messengerRepo, _, _ := setup(t)
	ctx := context.Background()
	messengerUserID := 123
	task := &models.Task{
		UserID:                 1,
		MessengerRelatedUserID: &messengerUserID,
		RequiresConfirmation:   false,
	}
	expectedErr := errors.New("messenger database error")

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), messengerUserID).Return(nil, expectedErr)

	id, childTaskID, err := service.CreateTask(ctx, task)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Equal(t, int64(0), childTaskID)
	assert.Contains(t, err.Error(), "messenger database error")
}

func TestTaskService_CreateTask_TaskRepositoryError(t *testing.T) {
	service, taskRepo, userRepo, _, _, _ := setup(t)
	ctx := context.Background()
	task := &models.Task{UserID: 1, RequiresConfirmation: false}
	expectedErr := errors.New("task creation failed")

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	taskRepo.EXPECT().CreateTask(gomock.Any(), task).Return(int64(0), expectedErr)

	id, childTaskID, err := service.CreateTask(ctx, task)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Equal(t, int64(0), childTaskID)
	assert.Contains(t, err.Error(), "task creation failed")
}

// GetTask Tests
func TestTaskService_GetTask_Success(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	expectedTask := &models.Task{
		ID:                   1,
		Title:                "Test Task",
		Description:          "Test Description",
		UserID:               1,
		Status:               "pending",
		RequiresConfirmation: false,
	}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), int64(1)).Return(expectedTask, nil)

	task, err := service.GetTask(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, expectedTask, task)
}

func TestTaskService_GetTask_NotFound(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), int64(1)).Return(nil, errs.ErrNotFound)

	task, err := service.GetTask(ctx, 1)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestTaskService_GetTask_RepositoryError(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	expectedErr := errors.New("database error")

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), int64(1)).Return(nil, expectedErr)

	task, err := service.GetTask(ctx, 1)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "database error")
}

// GetUserTasks Tests
func TestTaskService_GetUserTasks_Success(t *testing.T) {
	service, taskRepo, userRepo, _, _, _ := setup(t)
	ctx := context.Background()
	userID := int64(1)
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"
	expectedTasks := []*models.Task{
		{ID: 1, UserID: userID, Title: "Task 1", RequiresConfirmation: false},
		{ID: 2, UserID: userID, Title: "Task 2", RequiresConfirmation: false},
	}
	totalCount := 2

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
	taskRepo.EXPECT().GetTasksByUserIDWithPagination(gomock.Any(), userID, page, pageSize, orderBy, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).Return(expectedTasks, totalCount, nil)

	tasks, count, err := service.GetUserTasks(ctx, userID, page, pageSize, orderBy, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, expectedTasks, tasks)
	assert.Equal(t, totalCount, count)
}

func TestTaskService_GetUserTasks_EmptyList(t *testing.T) {
	service, taskRepo, userRepo, _, _, _ := setup(t)
	ctx := context.Background()
	userID := int64(1)
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"
	totalCount := 0

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
	taskRepo.EXPECT().GetTasksByUserIDWithPagination(gomock.Any(), userID, page, pageSize, orderBy, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).Return([]*models.Task{}, totalCount, nil)

	tasks, count, err := service.GetUserTasks(ctx, userID, page, pageSize, orderBy, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.NoError(t, err)
	assert.Empty(t, tasks)
	assert.Equal(t, totalCount, count)
}

func TestTaskService_GetUserTasks_UserNotFound(t *testing.T) {
	service, _, userRepo, _, _, _ := setup(t)
	ctx := context.Background()
	userID := int64(1)
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(nil, errs.ErrNotFound)

	tasks, count, err := service.GetUserTasks(ctx, userID, page, pageSize, orderBy, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

func TestTaskService_GetUserTasks_UserRepositoryError(t *testing.T) {
	service, _, userRepo, _, _, _ := setup(t)
	ctx := context.Background()
	userID := int64(1)
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"
	expectedErr := errors.New("user database error")

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(nil, expectedErr)

	tasks, count, err := service.GetUserTasks(ctx, userID, page, pageSize, orderBy, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "user database error")
}

func TestTaskService_GetUserTasks_TaskRepositoryError(t *testing.T) {
	service, taskRepo, userRepo, _, _, _ := setup(t)
	ctx := context.Background()
	userID := int64(1)
	page := 1
	pageSize := 50
	orderBy := "created_at DESC"
	expectedErr := errors.New("task database error")

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
	taskRepo.EXPECT().GetTasksByUserIDWithPagination(gomock.Any(), userID, page, pageSize, orderBy, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).Return(nil, 0, expectedErr)

	tasks, count, err := service.GetUserTasks(ctx, userID, page, pageSize, orderBy, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "task database error")
}

// UpdateTask Tests
func TestTaskService_UpdateTask_Success_AllFields(t *testing.T) {
	service, taskRepo, _, _, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	originalTask := &models.Task{
		ID:                   taskID,
		Title:                "Old Title",
		Description:          "Old Description",
		Status:               "pending",
		StartDate:            time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		RequiresConfirmation: false,
	}
	updateReq := &models.TaskUpdateRequest{
		Title:       ptrString("New Title"),
		Description: ptrString("New Description"),
		Status:      ptrString("done"),
		StartDate:   ptrTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(originalTask, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).Return(nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil).Times(2) // status change + general update

	updatedTask, err := service.UpdateTask(ctx, taskID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "New Title", updatedTask.Title)
	assert.Equal(t, "New Description", updatedTask.Description)
	assert.Equal(t, "done", updatedTask.Status)
	assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), updatedTask.StartDate)
}

func TestTaskService_UpdateTask_Success_PartialUpdate(t *testing.T) {
	service, taskRepo, _, _, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	originalTask := &models.Task{
		ID:                   taskID,
		Title:                "Original Title",
		Description:          "Original Description",
		Status:               "pending",
		StartDate:            time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		RequiresConfirmation: false,
	}
	updateReq := &models.TaskUpdateRequest{
		Title: ptrString("Updated Title"),
		// Only title is updated, other fields should remain unchanged
	}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(originalTask, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).Return(nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	updatedTask, err := service.UpdateTask(ctx, taskID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedTask.Title)
	assert.Equal(t, "Original Description", updatedTask.Description)
	assert.Equal(t, "pending", updatedTask.Status)
	assert.Equal(t, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), updatedTask.StartDate)
}

func TestTaskService_UpdateTask_TaskNotFound(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	updateReq := &models.TaskUpdateRequest{Title: ptrString("New Title")}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(nil, errs.ErrNotFound)

	task, err := service.UpdateTask(ctx, taskID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestTaskService_UpdateTask_GetTaskError(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	updateReq := &models.TaskUpdateRequest{Title: ptrString("New Title")}
	expectedErr := errors.New("database error")

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(nil, expectedErr)

	task, err := service.UpdateTask(ctx, taskID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_UpdateTask_UpdateError(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	originalTask := &models.Task{ID: taskID, Title: "Original Title", RequiresConfirmation: false}
	updateReq := &models.TaskUpdateRequest{Title: ptrString("New Title")}
	expectedErr := errors.New("update failed")

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(originalTask, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).Return(expectedErr)

	task, err := service.UpdateTask(ctx, taskID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "update failed")
}

// DeleteTask Tests
func TestTaskService_DeleteTask_Success(t *testing.T) {
	service, taskRepo, _, _, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	task := &models.Task{ID: taskID, Title: "Task to Delete", RequiresConfirmation: false}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(task, nil)
	taskRepo.EXPECT().GetDB().Return(nil).AnyTimes() // For transaction handling
	taskRepo.EXPECT().DeleteTask(gomock.Any(), taskID).Return(nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	err := service.DeleteTask(ctx, taskID)
	assert.NoError(t, err)
}

func TestTaskService_DeleteTask_TaskNotFound(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(nil, errs.ErrNotFound)

	err := service.DeleteTask(ctx, taskID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestTaskService_DeleteTask_GetTaskError(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	expectedErr := errors.New("database error")

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(nil, expectedErr)

	err := service.DeleteTask(ctx, taskID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_DeleteTask_DeleteError(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	task := &models.Task{ID: taskID, Title: "Task to Delete", RequiresConfirmation: false}
	expectedErr := errors.New("delete failed")

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(task, nil)
	taskRepo.EXPECT().GetDB().Return(nil).AnyTimes() // For transaction handling
	taskRepo.EXPECT().DeleteTask(gomock.Any(), taskID).Return(expectedErr)

	err := service.DeleteTask(ctx, taskID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete failed")
}

// NewTaskService Tests
func TestNewTaskService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	taskRepo := mock_repositories.NewMockTaskRepository(ctrl)
	userRepo := mock_repositories.NewMockUserRepository(ctrl)
	messengerRepo := mock_repositories.NewMockMessengerRepository(ctrl)
	taskHistoryRepo := mock_repositories.NewMockTaskHistoryRepository(ctrl)
	producer := &queue.Producer{}
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)

	service := NewTaskService(taskRepo, userRepo, messengerRepo, taskHistoryRepo, producer, testLogger)

	assert.NotNil(t, service)
	assert.Equal(t, taskRepo, service.taskRepo)
	assert.Equal(t, userRepo, service.userRepo)
	assert.Equal(t, messengerRepo, service.messengerRepo)
	assert.Equal(t, producer, service.producer)
}

// Edge Cases and Additional Tests
func TestTaskService_CreateTask_NilMessengerRelatedUserID(t *testing.T) {
	service, taskRepo, userRepo, _, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	task := &models.Task{
		UserID:                 1,
		Title:                  "Test Task",
		MessengerRelatedUserID: nil, // Should not call messenger repository
		RequiresConfirmation:   false,
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil)
	taskRepo.EXPECT().CreateTask(gomock.Any(), task).Return(int64(42), nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	id, childTaskID, err := service.CreateTask(ctx, task)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
	assert.Equal(t, int64(0), childTaskID)
}

func TestTaskService_UpdateTask_NilUpdateRequest(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	originalTask := &models.Task{ID: taskID, Title: "Original Title", RequiresConfirmation: false}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(originalTask, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), originalTask).Return(nil)

	task, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{})
	assert.NoError(t, err)
	assert.Equal(t, originalTask, task)
}

func TestTaskService_UpdateTask_AllNilFields(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	originalTask := &models.Task{
		ID:                   taskID,
		Title:                "Original Title",
		Description:          "Original Description",
		Status:               "pending",
		StartDate:            time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishDate:           ptrTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
		CronExpression:       ptrString("0 0 0 0 0"),
		RequiresConfirmation: false,
	}
	updateReq := &models.TaskUpdateRequest{
		Title:          nil,
		Description:    nil,
		Status:         nil,
		StartDate:      nil,
		FinishDate:     nil,
		CronExpression: nil,
	}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(originalTask, nil)
	taskRepo.EXPECT().GetDB().Return(nil).AnyTimes() // For transaction handling (parent task)
	taskRepo.EXPECT().GetChildTasksByParentID(gomock.Any(), taskID).Return([]*models.Task{}, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), originalTask).Return(nil)

	task, err := service.UpdateTask(ctx, taskID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, originalTask, task)
}

// GetTaskHistory Tests
func TestTaskService_GetTaskHistory_Success(t *testing.T) {
	service, taskRepo, _, _, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)
	expectedHistories := []*models.TaskHistory{
		{
			ID:        1,
			TaskID:    taskID,
			UserID:    1,
			Action:    string(models.TaskHistoryActionCreated),
			NewValue:  map[string]interface{}{"title": "Test Task"},
			CreatedAt: time.Now().UTC(),
		},
	}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(&models.Task{ID: taskID}, nil)
	taskHistoryRepo.EXPECT().GetTaskHistoryByTaskID(gomock.Any(), taskID).Return(expectedHistories, nil)

	histories, err := service.GetTaskHistory(ctx, taskID)
	assert.NoError(t, err)
	assert.Equal(t, expectedHistories, histories)
}

func TestTaskService_GetTaskHistory_TaskNotFound(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(nil, errs.ErrNotFound)

	histories, err := service.GetTaskHistory(ctx, taskID)
	assert.Error(t, err)
	assert.Nil(t, histories)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

// GetUserTaskHistory Tests
func TestTaskService_GetUserTaskHistory_Success(t *testing.T) {
	service, _, userRepo, _, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	userID := int64(1)
	limit := 10
	offset := 0
	expectedHistories := []*models.TaskHistory{
		{
			ID:        1,
			TaskID:    1,
			UserID:    userID,
			Action:    string(models.TaskHistoryActionCreated),
			NewValue:  map[string]interface{}{"title": "Test Task"},
			CreatedAt: time.Now().UTC(),
		},
	}

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(&models.User{ID: userID}, nil)
	taskHistoryRepo.EXPECT().GetTaskHistoryByUserID(gomock.Any(), userID, limit, offset).Return(expectedHistories, nil)

	histories, err := service.GetUserTaskHistory(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Equal(t, expectedHistories, histories)
}

func TestTaskService_GetUserTaskHistory_UserNotFound(t *testing.T) {
	service, _, userRepo, _, _, _ := setup(t)
	ctx := context.Background()
	userID := int64(1)

	userRepo.EXPECT().GetUserByID(gomock.Any(), userID).Return(nil, errs.ErrNotFound)

	histories, err := service.GetUserTaskHistory(ctx, userID, 10, 0)
	assert.Error(t, err)
	assert.Nil(t, histories)
	assert.Contains(t, err.Error(), "unprocessable entity")
}
