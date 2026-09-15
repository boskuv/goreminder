package service

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/attachments"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTaskService_QueueTask_Schedule_Success(t *testing.T) {
	service, taskRepo, _, messengerRepo, _, _ := setup(t)
	ctx := context.Background()
	mu := 5
	messengerID := int64(1)
	task := &models.Task{
		ID:                     42,
		UserID:                 1,
		Title:                  "queued",
		Description:            "desc",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(time.Hour),
		MessengerRelatedUserID: &mu,
		RequiresConfirmation:   false,
	}
	pub := &stubPublisher{}
	service.producer = pub

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), int64(42)).Return(task, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "chat-1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)

	err := service.QueueTask(ctx, &models.ScheduledTask{
		Action: models.ScheduledTaskActionSchedule,
		TaskID: 42,
	})
	require.NoError(t, err)
	require.Len(t, pub.published, 1)
}

func TestTaskService_QueueTask_Delete_Success(t *testing.T) {
	service, taskRepo, _, messengerRepo, _, _ := setup(t)
	ctx := context.Background()
	mu := 5
	messengerID := int64(1)
	task := &models.Task{
		ID:                     42,
		UserID:                 1,
		Title:                  "queued",
		Status:                 string(models.TaskStatusScheduled),
		MessengerRelatedUserID: &mu,
	}
	pub := &stubPublisher{}
	service.producer = pub

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), int64(42)).Return(task, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "chat-1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)

	err := service.QueueTask(ctx, &models.ScheduledTask{
		Action: models.ScheduledTaskActionDelete,
		TaskID: 42,
	})
	require.NoError(t, err)
	require.Len(t, pub.published, 1)
}

func TestTaskService_QueueTask_UnsupportedAction(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), int64(1)).Return(&models.Task{ID: 1}, nil)

	err := service.QueueTask(ctx, &models.ScheduledTask{Action: "boom", TaskID: 1})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrValidation))
}

func TestTaskService_QueueTask_TaskNotFound(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), int64(99)).Return(nil, errs.ErrNotFound)

	err := service.QueueTask(ctx, &models.ScheduledTask{
		Action: models.ScheduledTaskActionSchedule,
		TaskID: 99,
	})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound))
}

func TestTaskService_QueueTask_PublishError(t *testing.T) {
	service, taskRepo, _, messengerRepo, _, _ := setup(t)
	ctx := context.Background()
	mu := 5
	messengerID := int64(1)
	task := &models.Task{
		ID:                     42,
		UserID:                 1,
		Title:                  "queued",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(time.Hour),
		MessengerRelatedUserID: &mu,
	}
	service.producer = &stubPublisher{err: errors.New("amqp down")}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), int64(42)).Return(task, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "chat-1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)

	err := service.QueueTask(ctx, &models.ScheduledTask{
		Action: models.ScheduledTaskActionSchedule,
		TaskID: 42,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amqp down")
}

func TestTaskService_MarkTaskAsDone_AlreadyDone(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	task := &models.Task{ID: 1, Status: string(models.TaskStatusDone)}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), int64(1)).Return(task, nil)

	out, err := service.MarkTaskAsDone(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, task, out)
}

func TestTaskService_MarkTaskAsDone_NotFound(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), int64(1)).Return(nil, errs.ErrNotFound)

	out, err := service.MarkTaskAsDone(ctx, 1)
	assert.Nil(t, out)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound))
}

func TestTaskService_MarkTaskAsDone_NoDB(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	mu := 3
	task := &models.Task{
		ID:                     1,
		Status:                 string(models.TaskStatusScheduled),
		MessengerRelatedUserID: &mu,
	}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), int64(1)).Return(task, nil)
	taskRepo.EXPECT().GetDB().Return(nil)

	out, err := service.MarkTaskAsDone(ctx, 1)
	assert.Nil(t, out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection not available")
}

func TestTaskService_MarkTaskAsDone_Success(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	mu := 3
	messengerID := int64(9)
	task := &models.Task{
		ID:                     1,
		UserID:                 10,
		Title:                  "done me",
		Status:                 string(models.TaskStatusScheduled),
		MessengerRelatedUserID: &mu,
		RequiresConfirmation:   false,
	}

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mockDB.ExpectBegin()
	mockDB.ExpectCommit()

	pub := &stubPublisher{}
	service.producer = pub

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), int64(1)).Return(task, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().UpdateTaskWithTx(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *sqlx.Tx, updated *models.Task) error {
			assert.Equal(t, string(models.TaskStatusDone), updated.Status)
			assert.NotNil(t, updated.FinishDate)
			return nil
		},
	)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	out, err := service.MarkTaskAsDone(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, string(models.TaskStatusDone), out.Status)
	require.Len(t, pub.published, 1)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_MarkTaskAsDone_RollbackOnPublishError(t *testing.T) {
	service, taskRepo, _, messengerRepo, _, _ := setup(t)
	ctx := context.Background()
	mu := 3
	messengerID := int64(9)
	task := &models.Task{
		ID:                     1,
		UserID:                 10,
		Status:                 string(models.TaskStatusScheduled),
		MessengerRelatedUserID: &mu,
	}

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mockDB.ExpectBegin()
	mockDB.ExpectRollback()

	service.producer = &stubPublisher{err: errors.New("publish failed")}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), int64(1)).Return(task, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().UpdateTaskWithTx(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID,
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)

	out, err := service.MarkTaskAsDone(ctx, 1)
	assert.Nil(t, out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to queue delete_task message")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_GetAllTasks_Success(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	status := string(models.TaskStatusScheduled)
	expected := []*models.Task{{ID: 1, Title: "a"}, {ID: 2, Title: "b"}}

	taskRepo.EXPECT().GetAllTasks(
		gomock.Any(), 1, 20, "id DESC",
		&status, nil, nil, nil, nil, nil, nil, nil, nil,
	).Return(expected, 2, nil)

	tasks, total, err := service.GetAllTasks(ctx, 1, 20, "id DESC", &status, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Equal(t, expected, tasks)
}

func TestTaskService_GetAllTasks_RepoError(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()

	taskRepo.EXPECT().GetAllTasks(
		gomock.Any(), 1, 10, "created_at DESC",
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
	).Return(nil, 0, errors.New("db error"))

	tasks, total, err := service.GetAllTasks(ctx, 1, 10, "created_at DESC", nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.Nil(t, tasks)
	assert.Equal(t, 0, total)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestTaskService_RescheduleTasks_ContinuesOnPartialFailure(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	mu := 7
	pub := &stubPublisher{}
	service.producer = pub

	okTask := &models.Task{
		ID:                     1,
		UserID:                 1,
		Title:                  "ok",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(-24 * time.Hour),
		MessengerRelatedUserID: &mu,
		Muted:                  true,
	}
	badTask := &models.Task{
		ID:                     2,
		UserID:                 1,
		Title:                  "bad",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(-24 * time.Hour),
		MessengerRelatedUserID: &mu,
		Muted:                  true,
	}

	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: ptrInt64(1), ChatID: "c1",
	}, nil).Times(2)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), int64(1)).Return(&models.Messenger{ID: 1, Name: "telegram"}, nil).Times(2)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, task *models.Task) error {
		if task.ID == 2 {
			return errors.New("update failed")
		}
		return nil
	}).Times(2)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	err := service.RescheduleTasks(ctx, []*models.Task{okTask, badTask})
	assert.NoError(t, err, "RescheduleTasks aggregates failures and still returns nil")
}

func TestTaskService_AttachmentHistoryMetaFromModel(t *testing.T) {
	assert.Equal(t, AttachmentHistoryMeta{}, AttachmentHistoryMetaFromModel(nil))

	meta := AttachmentHistoryMetaFromModel(&attachments.Attachment{
		ID:           "att-1",
		OriginalName: "file.txt",
		ContentType:  "text/plain",
		SizeBytes:    12,
	})
	assert.Equal(t, "att-1", meta.AttachmentID)
	assert.Equal(t, "file.txt", meta.OriginalName)
	assert.Equal(t, "text/plain", meta.ContentType)
	assert.Equal(t, int64(12), meta.SizeBytes)
}

func TestTaskService_MarkTaskAsDone_ChildCreatesNextOccurrence(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	parentID := int64(100)
	taskID := int64(101)
	mu := 5
	messengerID := int64(1)
	pub := &stubPublisher{}
	service.producer = pub

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectCommit()

	child := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "child",
		Description:            "d",
		Status:                 string(models.TaskStatusScheduled),
		ParentID:               &parentID,
		MessengerRelatedUserID: &mu,
		RequiresConfirmation:   true,
	}
	parent := &models.Task{
		ID:                     parentID,
		UserID:                 1,
		Title:                  "parent",
		Description:            "d",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(-time.Hour),
		CronExpression:         ptrString("0 9 * * *"),
		RequiresConfirmation:   true,
		MessengerRelatedUserID: &mu,
	}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(child, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().UpdateTaskWithTx(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil).AnyTimes()
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil).AnyTimes()
	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), parentID).Return(parent, nil)
	taskRepo.EXPECT().CreateTask(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, next *models.Task) (int64, error) {
		assert.Equal(t, parentID, *next.ParentID)
		assert.Nil(t, next.CronExpression)
		return int64(102), nil
	})
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	out, err := service.MarkTaskAsDone(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, string(models.TaskStatusDone), out.Status)
	assert.GreaterOrEqual(t, len(pub.published), 1)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_MarkTaskAsDone_ParentMarksActiveChildren(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(200)
	mu := 5
	messengerID := int64(1)
	pub := &stubPublisher{}
	service.producer = pub

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectCommit()
	mockDB.ExpectBegin()
	mockDB.ExpectCommit()

	parent := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "parent",
		Description:            "d",
		Status:                 string(models.TaskStatusScheduled),
		CronExpression:         ptrString("0 9 * * *"),
		RequiresConfirmation:   true,
		MessengerRelatedUserID: &mu,
	}
	active := &models.Task{
		ID: 201, Title: "parent", Description: "d", UserID: 1, ParentID: &taskID,
		Status: string(models.TaskStatusScheduled), MessengerRelatedUserID: &mu,
	}
	done := &models.Task{ID: 202, Status: string(models.TaskStatusDone), ParentID: &taskID}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(parent, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB).Times(2)
	taskRepo.EXPECT().UpdateTaskWithTx(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).MinTimes(2)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil).AnyTimes()
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil).AnyTimes()
	taskRepo.EXPECT().GetChildTasksByParentID(gomock.Any(), taskID).Return([]*models.Task{active, done}, nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	out, err := service.MarkTaskAsDone(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, string(models.TaskStatusDone), out.Status)
	assert.GreaterOrEqual(t, len(pub.published), 2)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_MarkTaskAsDone_PurgesAttachmentsWhenEnabled(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	service.attachmentsPurgeOnTaskDone = true
	ctx := context.Background()
	taskID := int64(300)
	mu := 5
	messengerID := int64(1)
	pub := &stubPublisher{}
	service.producer = pub

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectCommit()

	task := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "t",
		Status:                 string(models.TaskStatusScheduled),
		MessengerRelatedUserID: &mu,
	}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(task, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().UpdateTaskWithTx(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)
	taskRepo.EXPECT().GetChildTasksByParentID(gomock.Any(), taskID).Return([]*models.Task{}, nil)

	out, err := service.MarkTaskAsDone(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, string(models.TaskStatusDone), out.Status)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}
