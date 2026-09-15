package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	mock_repositories "github.com/boskuv/goreminder/internal/mocks/repository"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/attachments"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newSchedulerForTest(t *testing.T) (*TaskScheduler, *mock_repositories.MockTaskRepository, *TaskService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	taskRepo := mock_repositories.NewMockTaskRepository(ctrl)
	userRepo := mock_repositories.NewMockUserRepository(ctrl)
	messengerRepo := mock_repositories.NewMockMessengerRepository(ctrl)
	taskHistoryRepo := mock_repositories.NewMockTaskHistoryRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	taskSvc := NewTaskService(taskRepo, userRepo, messengerRepo, taskHistoryRepo, &stubPublisher{}, attachments.NewNoopClient(), false, testLogger)
	scheduler := NewTaskScheduler(taskRepo, taskSvc, testLogger)
	return scheduler, taskRepo, taskSvc
}

func TestNewTaskScheduler(t *testing.T) {
	scheduler, taskRepo, taskSvc := newSchedulerForTest(t)
	assert.NotNil(t, scheduler)
	assert.Equal(t, taskRepo, scheduler.taskRepo)
	assert.Equal(t, taskSvc, scheduler.taskSvc)
	assert.NotNil(t, scheduler.scheduler)
}

func TestTaskScheduler_RunScheduledRescheduling_NoTasks(t *testing.T) {
	scheduler, taskRepo, _ := newSchedulerForTest(t)
	ctx := context.Background()

	// Empty one-time list returns early; cron pass is skipped by current implementation.
	taskRepo.EXPECT().GetTasksNeedingRescheduling(gomock.Any()).Return([]*models.Task{}, nil)

	err := scheduler.RunScheduledRescheduling(ctx)
	assert.NoError(t, err)
}

func TestTaskScheduler_RunScheduledRescheduling_GetTasksError(t *testing.T) {
	scheduler, taskRepo, _ := newSchedulerForTest(t)
	ctx := context.Background()

	taskRepo.EXPECT().GetTasksNeedingRescheduling(gomock.Any()).Return(nil, errors.New("db error"))

	err := scheduler.RunScheduledRescheduling(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestTaskScheduler_RunScheduledRescheduling_ReschedulesTasks(t *testing.T) {
	ctrl := gomock.NewController(t)
	taskRepo := mock_repositories.NewMockTaskRepository(ctrl)
	userRepo := mock_repositories.NewMockUserRepository(ctrl)
	messengerRepo := mock_repositories.NewMockMessengerRepository(ctrl)
	taskHistoryRepo := mock_repositories.NewMockTaskHistoryRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	taskSvc := NewTaskService(taskRepo, userRepo, messengerRepo, taskHistoryRepo, &stubPublisher{}, attachments.NewNoopClient(), false, testLogger)
	scheduler := NewTaskScheduler(taskRepo, taskSvc, testLogger)

	ctx := context.Background()
	mu := 7
	oldStart := time.Now().UTC().Add(-48 * time.Hour)
	task := &models.Task{
		ID:                     1,
		UserID:                 1,
		Title:                  "one-time",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              oldStart,
		MessengerRelatedUserID: &mu,
		Muted:                  true,
		RequiresConfirmation:   false,
	}
	cronExpr := "0 9 * * *"
	cronTask := &models.Task{
		ID:                   2,
		UserID:               1,
		Title:                "cron",
		Status:               string(models.TaskStatusScheduled),
		StartDate:            oldStart,
		CronExpression:       &cronExpr,
		RequiresConfirmation: false,
	}

	taskRepo.EXPECT().GetTasksNeedingRescheduling(gomock.Any()).Return([]*models.Task{task}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: ptrInt64(1), ChatID: "c1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), int64(1)).Return(&models.Messenger{ID: 1, Name: "telegram"}, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).Return(nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	taskRepo.EXPECT().GetTasksWithCronNeedingRescheduling(gomock.Any()).Return([]*models.Task{cronTask}, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, updated *models.Task) error {
		assert.Equal(t, int64(2), updated.ID)
		assert.True(t, updated.StartDate.After(oldStart))
		return nil
	})

	err := scheduler.RunScheduledRescheduling(ctx)
	require.NoError(t, err)
}

func TestTaskScheduler_RunScheduledRescheduling_CronFetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	taskRepo := mock_repositories.NewMockTaskRepository(ctrl)
	userRepo := mock_repositories.NewMockUserRepository(ctrl)
	messengerRepo := mock_repositories.NewMockMessengerRepository(ctrl)
	taskHistoryRepo := mock_repositories.NewMockTaskHistoryRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	taskSvc := NewTaskService(taskRepo, userRepo, messengerRepo, taskHistoryRepo, &stubPublisher{}, attachments.NewNoopClient(), false, testLogger)
	scheduler := NewTaskScheduler(taskRepo, taskSvc, testLogger)

	ctx := context.Background()
	mu := 7
	task := &models.Task{
		ID:                     1,
		UserID:                 1,
		Title:                  "one-time",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(-48 * time.Hour),
		MessengerRelatedUserID: &mu,
		Muted:                  true,
		RequiresConfirmation:   false,
	}

	taskRepo.EXPECT().GetTasksNeedingRescheduling(gomock.Any()).Return([]*models.Task{task}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: ptrInt64(1), ChatID: "c1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), int64(1)).Return(&models.Messenger{ID: 1, Name: "telegram"}, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).Return(nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)
	taskRepo.EXPECT().GetTasksWithCronNeedingRescheduling(gomock.Any()).Return(nil, errors.New("cron query failed"))

	err := scheduler.RunScheduledRescheduling(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cron query failed")
}

func TestTaskScheduler_StartScheduler_StopsOnCancel(t *testing.T) {
	scheduler, _, _ := newSchedulerForTest(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		scheduler.StartScheduler(ctx, "23:59")
		close(done)
	}()

	// Give the scheduler a moment to start, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartScheduler did not stop after context cancel")
	}
}
