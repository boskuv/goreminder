package service

import (
	"errors"
	"testing"
	"time"

	errs "github.com/boskuv/goreminder/internal/errors"
	mock_repositories "github.com/boskuv/goreminder/internal/mocks/repository"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/queue"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setup(t *testing.T) (*TaskService, *mock_repositories.MockTaskRepository, *mock_repositories.MockUserRepository, *mock_repositories.MockMessengerRepository, *queue.Producer) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	taskRepo := mock_repositories.NewMockTaskRepository(ctrl)
	userRepo := mock_repositories.NewMockUserRepository(ctrl)
	messengerRepo := mock_repositories.NewMockMessengerRepository(ctrl)
	producer := &queue.Producer{}

	service := NewTaskService(taskRepo, userRepo, messengerRepo, producer)
	return service, taskRepo, userRepo, messengerRepo, producer
}

// Helper functions
func ptrString(s string) *string     { return &s }
func ptrTime(t time.Time) *time.Time { return &t }
func ptrInt(i int) *int              { return &i }

// CreateTask Tests
func TestTaskService_CreateTask_Success(t *testing.T) {
	service, taskRepo, userRepo, _, _ := setup(t)
	task := &models.Task{
		UserID:      1,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      "pending",
	}

	userRepo.EXPECT().GetUserByID(int64(1)).Return(&models.User{ID: 1}, nil)
	taskRepo.EXPECT().CreateTask(task).Return(int64(42), nil)

	id, err := service.CreateTask(task)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestTaskService_CreateTask_WithMessengerRelatedUser_Success(t *testing.T) {
	service, taskRepo, userRepo, messengerRepo, _ := setup(t)
	messengerUserID := 123
	task := &models.Task{
		UserID:                 1,
		Title:                  "Test Task",
		Description:            "Test Description",
		Status:                 "pending",
		MessengerRelatedUserID: &messengerUserID,
	}

	userRepo.EXPECT().GetUserByID(int64(1)).Return(&models.User{ID: 1}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(messengerUserID).Return(&models.MessengerRelatedUser{ID: int64(messengerUserID)}, nil)
	taskRepo.EXPECT().CreateTask(task).Return(int64(42), nil)

	id, err := service.CreateTask(task)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestTaskService_CreateTask_UserNotFound(t *testing.T) {
	service, _, userRepo, _, _ := setup(t)
	task := &models.Task{UserID: 1}

	userRepo.EXPECT().GetUserByID(int64(1)).Return(nil, errs.ErrNotFound)

	id, err := service.CreateTask(task)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

func TestTaskService_CreateTask_UserRepositoryError(t *testing.T) {
	service, _, userRepo, _, _ := setup(t)
	task := &models.Task{UserID: 1}
	expectedErr := errors.New("database error")

	userRepo.EXPECT().GetUserByID(int64(1)).Return(nil, expectedErr)

	id, err := service.CreateTask(task)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_CreateTask_MessengerRelatedUserNotFound(t *testing.T) {
	service, _, userRepo, messengerRepo, _ := setup(t)
	messengerUserID := 123
	task := &models.Task{
		UserID:                 1,
		MessengerRelatedUserID: &messengerUserID,
	}

	userRepo.EXPECT().GetUserByID(int64(1)).Return(&models.User{ID: 1}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(messengerUserID).Return(nil, errs.ErrNotFound)

	id, err := service.CreateTask(task)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

func TestTaskService_CreateTask_MessengerRepositoryError(t *testing.T) {
	service, _, userRepo, messengerRepo, _ := setup(t)
	messengerUserID := 123
	task := &models.Task{
		UserID:                 1,
		MessengerRelatedUserID: &messengerUserID,
	}
	expectedErr := errors.New("messenger database error")

	userRepo.EXPECT().GetUserByID(int64(1)).Return(&models.User{ID: 1}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(messengerUserID).Return(nil, expectedErr)

	id, err := service.CreateTask(task)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "messenger database error")
}

func TestTaskService_CreateTask_TaskRepositoryError(t *testing.T) {
	service, taskRepo, userRepo, _, _ := setup(t)
	task := &models.Task{UserID: 1}
	expectedErr := errors.New("task creation failed")

	userRepo.EXPECT().GetUserByID(int64(1)).Return(&models.User{ID: 1}, nil)
	taskRepo.EXPECT().CreateTask(task).Return(int64(0), expectedErr)

	id, err := service.CreateTask(task)
	assert.Error(t, err)
	assert.Equal(t, int64(0), id)
	assert.Contains(t, err.Error(), "task creation failed")
}

// GetTask Tests
func TestTaskService_GetTask_Success(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	expectedTask := &models.Task{
		ID:          1,
		Title:       "Test Task",
		Description: "Test Description",
		UserID:      1,
		Status:      "pending",
	}

	taskRepo.EXPECT().GetTaskByID(int64(1)).Return(expectedTask, nil)

	task, err := service.GetTask(1)
	assert.NoError(t, err)
	assert.Equal(t, expectedTask, task)
}

func TestTaskService_GetTask_NotFound(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)

	taskRepo.EXPECT().GetTaskByID(int64(1)).Return(nil, errs.ErrNotFound)

	task, err := service.GetTask(1)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestTaskService_GetTask_RepositoryError(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	expectedErr := errors.New("database error")

	taskRepo.EXPECT().GetTaskByID(int64(1)).Return(nil, expectedErr)

	task, err := service.GetTask(1)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "database error")
}

// GetUserTasks Tests
func TestTaskService_GetUserTasks_Success(t *testing.T) {
	service, taskRepo, userRepo, _, _ := setup(t)
	userID := int64(1)
	expectedTasks := []*models.Task{
		{ID: 1, UserID: userID, Title: "Task 1"},
		{ID: 2, UserID: userID, Title: "Task 2"},
	}

	userRepo.EXPECT().GetUserByID(userID).Return(&models.User{ID: userID}, nil)
	taskRepo.EXPECT().GetTasksByUserID(userID).Return(expectedTasks, nil)

	tasks, err := service.GetUserTasks(userID)
	assert.NoError(t, err)
	assert.Equal(t, expectedTasks, tasks)
}

func TestTaskService_GetUserTasks_EmptyList(t *testing.T) {
	service, taskRepo, userRepo, _, _ := setup(t)
	userID := int64(1)

	userRepo.EXPECT().GetUserByID(userID).Return(&models.User{ID: userID}, nil)
	taskRepo.EXPECT().GetTasksByUserID(userID).Return([]*models.Task{}, nil)

	tasks, err := service.GetUserTasks(userID)
	assert.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestTaskService_GetUserTasks_UserNotFound(t *testing.T) {
	service, _, userRepo, _, _ := setup(t)
	userID := int64(1)

	userRepo.EXPECT().GetUserByID(userID).Return(nil, errs.ErrNotFound)

	tasks, err := service.GetUserTasks(userID)
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "unprocessable entity")
}

func TestTaskService_GetUserTasks_UserRepositoryError(t *testing.T) {
	service, _, userRepo, _, _ := setup(t)
	userID := int64(1)
	expectedErr := errors.New("user database error")

	userRepo.EXPECT().GetUserByID(userID).Return(nil, expectedErr)

	tasks, err := service.GetUserTasks(userID)
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "user database error")
}

func TestTaskService_GetUserTasks_TaskRepositoryError(t *testing.T) {
	service, taskRepo, userRepo, _, _ := setup(t)
	userID := int64(1)
	expectedErr := errors.New("task database error")

	userRepo.EXPECT().GetUserByID(userID).Return(&models.User{ID: userID}, nil)
	taskRepo.EXPECT().GetTasksByUserID(userID).Return(nil, expectedErr)

	tasks, err := service.GetUserTasks(userID)
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "task database error")
}

// UpdateTask Tests
func TestTaskService_UpdateTask_Success_AllFields(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)
	originalTask := &models.Task{
		ID:          taskID,
		Title:       "Old Title",
		Description: "Old Description",
		Status:      "pending",
		StartDate:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	updateReq := &models.TaskUpdateRequest{
		Title:       ptrString("New Title"),
		Description: ptrString("New Description"),
		Status:      ptrString("completed"),
		StartDate:   ptrTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	taskRepo.EXPECT().GetTaskByID(taskID).Return(originalTask, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any()).Return(nil)

	updatedTask, err := service.UpdateTask(taskID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "New Title", updatedTask.Title)
	assert.Equal(t, "New Description", updatedTask.Description)
	assert.Equal(t, "completed", updatedTask.Status)
	assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), updatedTask.StartDate)
}

func TestTaskService_UpdateTask_Success_PartialUpdate(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)
	originalTask := &models.Task{
		ID:          taskID,
		Title:       "Original Title",
		Description: "Original Description",
		Status:      "pending",
		StartDate:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	updateReq := &models.TaskUpdateRequest{
		Title: ptrString("Updated Title"),
		// Only title is updated, other fields should remain unchanged
	}

	taskRepo.EXPECT().GetTaskByID(taskID).Return(originalTask, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any()).Return(nil)

	updatedTask, err := service.UpdateTask(taskID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedTask.Title)
	assert.Equal(t, "Original Description", updatedTask.Description)
	assert.Equal(t, "pending", updatedTask.Status)
	assert.Equal(t, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), updatedTask.StartDate)
}

func TestTaskService_UpdateTask_TaskNotFound(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)
	updateReq := &models.TaskUpdateRequest{Title: ptrString("New Title")}

	taskRepo.EXPECT().GetTaskByID(taskID).Return(nil, errs.ErrNotFound)

	task, err := service.UpdateTask(taskID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestTaskService_UpdateTask_GetTaskError(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)
	updateReq := &models.TaskUpdateRequest{Title: ptrString("New Title")}
	expectedErr := errors.New("database error")

	taskRepo.EXPECT().GetTaskByID(taskID).Return(nil, expectedErr)

	task, err := service.UpdateTask(taskID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_UpdateTask_UpdateError(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)
	originalTask := &models.Task{ID: taskID, Title: "Original Title"}
	updateReq := &models.TaskUpdateRequest{Title: ptrString("New Title")}
	expectedErr := errors.New("update failed")

	taskRepo.EXPECT().GetTaskByID(taskID).Return(originalTask, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any()).Return(expectedErr)

	task, err := service.UpdateTask(taskID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "update failed")
}

// DeleteTask Tests
func TestTaskService_DeleteTask_Success(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)
	task := &models.Task{ID: taskID, Title: "Task to Delete"}

	taskRepo.EXPECT().GetTaskByID(taskID).Return(task, nil)
	taskRepo.EXPECT().DeleteTask(taskID).Return(nil)

	err := service.DeleteTask(taskID)
	assert.NoError(t, err)
}

func TestTaskService_DeleteTask_TaskNotFound(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)

	taskRepo.EXPECT().GetTaskByID(taskID).Return(nil, errs.ErrNotFound)

	err := service.DeleteTask(taskID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data found matching criteria")
}

func TestTaskService_DeleteTask_GetTaskError(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)
	expectedErr := errors.New("database error")

	taskRepo.EXPECT().GetTaskByID(taskID).Return(nil, expectedErr)

	err := service.DeleteTask(taskID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestTaskService_DeleteTask_DeleteError(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)
	task := &models.Task{ID: taskID, Title: "Task to Delete"}
	expectedErr := errors.New("delete failed")

	taskRepo.EXPECT().GetTaskByID(taskID).Return(task, nil)
	taskRepo.EXPECT().DeleteTask(taskID).Return(expectedErr)

	err := service.DeleteTask(taskID)
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
	producer := &queue.Producer{}

	service := NewTaskService(taskRepo, userRepo, messengerRepo, producer)

	assert.NotNil(t, service)
	assert.Equal(t, taskRepo, service.taskRepo)
	assert.Equal(t, userRepo, service.userRepo)
	assert.Equal(t, messengerRepo, service.messengerRepo)
	assert.Equal(t, producer, service.producer)
}

// Edge Cases and Additional Tests
func TestTaskService_CreateTask_NilMessengerRelatedUserID(t *testing.T) {
	service, taskRepo, userRepo, _, _ := setup(t)
	task := &models.Task{
		UserID:                 1,
		Title:                  "Test Task",
		MessengerRelatedUserID: nil, // Should not call messenger repository
	}

	userRepo.EXPECT().GetUserByID(int64(1)).Return(&models.User{ID: 1}, nil)
	taskRepo.EXPECT().CreateTask(task).Return(int64(42), nil)

	id, err := service.CreateTask(task)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestTaskService_UpdateTask_NilUpdateRequest(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)
	originalTask := &models.Task{ID: taskID, Title: "Original Title"}

	taskRepo.EXPECT().GetTaskByID(taskID).Return(originalTask, nil)
	taskRepo.EXPECT().UpdateTask(originalTask).Return(nil)

	task, err := service.UpdateTask(taskID, &models.TaskUpdateRequest{})
	assert.NoError(t, err)
	assert.Equal(t, originalTask, task)
}

func TestTaskService_UpdateTask_AllNilFields(t *testing.T) {
	service, taskRepo, _, _, _ := setup(t)
	taskID := int64(1)
	originalTask := &models.Task{
		ID:             taskID,
		Title:          "Original Title",
		Description:    "Original Description",
		Status:         "pending",
		StartDate:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		FinishDate:     ptrTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
		CronExpression: ptrString("0 0 0 0 0"),
	}
	updateReq := &models.TaskUpdateRequest{
		Title:          nil,
		Description:    nil,
		Status:         nil,
		StartDate:      nil,
		FinishDate:     nil,
		CronExpression: nil,
	}

	taskRepo.EXPECT().GetTaskByID(taskID).Return(originalTask, nil)
	taskRepo.EXPECT().UpdateTask(originalTask).Return(nil)

	task, err := service.UpdateTask(taskID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, originalTask, task)
}
