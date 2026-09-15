package service

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTaskService_UpdateTask_InvalidStatus(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(&models.Task{
		ID: taskID, UserID: 1, Title: "t", Status: string(models.TaskStatusScheduled),
	}, nil)

	out, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{
		Status: ptrString("not-a-status"),
	})
	assert.Nil(t, out)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrValidation))
}

func TestTaskService_UpdateTask_InvalidRRule(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(1)

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(&models.Task{
		ID: taskID, UserID: 1, Title: "t",
		Status:    string(models.TaskStatusScheduled),
		StartDate: time.Now().UTC().Add(time.Hour),
	}, nil)

	out, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{
		RRule: ptrString("FREQ=NOTAREALFREQ"),
	})
	assert.Nil(t, out)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrValidation))
}

func TestTaskService_UpdateTask_StatusToDeleted_PublishesDelete(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(10)
	mu := 5
	messengerID := int64(1)
	pub := &stubPublisher{}
	service.producer = pub

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(&models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "t",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(2 * time.Hour),
		MessengerRelatedUserID: &mu,
	}, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	out, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{
		Status: ptrString(string(models.TaskStatusDeleted)),
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, string(models.TaskStatusDeleted), out.Status)
	require.Len(t, pub.published, 1)
}

func TestTaskService_UpdateTask_Mute_PublishesDelete(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(11)
	mu := 5
	messengerID := int64(1)
	pub := &stubPublisher{}
	service.producer = pub

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(&models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "t",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(2 * time.Hour),
		MessengerRelatedUserID: &mu,
		Muted:                  false,
	}, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	out, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{
		Muted: ptrBool(true),
	})
	require.NoError(t, err)
	assert.True(t, out.Muted)
	require.Len(t, pub.published, 1)
}

func TestTaskService_UpdateTask_Unmute_PublishesSchedule(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(12)
	mu := 5
	messengerID := int64(1)
	pub := &stubPublisher{}
	service.producer = pub

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(&models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "t",
		Description:            "d",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(2 * time.Hour),
		MessengerRelatedUserID: &mu,
		Muted:                  true,
	}, nil)
	taskRepo.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil).AnyTimes()
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil).AnyTimes()
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	out, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{
		Muted: ptrBool(false),
	})
	require.NoError(t, err)
	assert.False(t, out.Muted)
	require.NotEmpty(t, pub.published)
}

func TestTaskService_UpdateTask_RequiresConfirmationRemoved_DeletesChildren(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(20)
	mu := 7
	messengerID := int64(3)
	pub := &stubPublisher{}
	service.producer = pub

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectCommit()

	parent := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "parent",
		Description:            "d",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(24 * time.Hour),
		CronExpression:         ptrString("0 9 * * *"),
		RequiresConfirmation:   true,
		MessengerRelatedUserID: &mu,
	}
	child := &models.Task{ID: 201, Status: string(models.TaskStatusScheduled), ParentID: &taskID}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(parent, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().UpdateTaskWithTx(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	taskRepo.EXPECT().GetChildTasksByParentID(gomock.Any(), taskID).Return([]*models.Task{child}, nil)
	taskRepo.EXPECT().DeleteChildTasksWithTx(gomock.Any(), gomock.Any(), taskID).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil).AnyTimes()
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil).AnyTimes()
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	out, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{
		RequiresConfirmation: ptrBool(false),
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.RequiresConfirmation)
	assert.GreaterOrEqual(t, len(pub.published), 1)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_UpdateTask_RequiresConfirmationRemoved_RollbackOnPublishError(t *testing.T) {
	service, taskRepo, _, messengerRepo, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(21)
	mu := 7
	messengerID := int64(3)
	service.producer = &stubPublisher{err: errors.New("publish failed")}

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectRollback()

	parent := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "parent",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(24 * time.Hour),
		CronExpression:         ptrString("0 9 * * *"),
		RequiresConfirmation:   true,
		MessengerRelatedUserID: &mu,
	}
	child := &models.Task{ID: 211, Status: string(models.TaskStatusPending), ParentID: &taskID}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(parent, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().UpdateTaskWithTx(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	taskRepo.EXPECT().GetChildTasksByParentID(gomock.Any(), taskID).Return([]*models.Task{child}, nil)
	taskRepo.EXPECT().DeleteChildTasksWithTx(gomock.Any(), gomock.Any(), taskID).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)

	out, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{
		RequiresConfirmation: ptrBool(false),
	})
	assert.Nil(t, out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to queue delete_task message for child task")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_UpdateTask_RequiresConfirmationAdded_CreatesChild(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(22)
	mu := 7
	messengerID := int64(3)
	pub := &stubPublisher{}
	service.producer = pub

	parent := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "recurring",
		Description:            "d",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(-time.Hour),
		CronExpression:         ptrString("0 9 * * *"),
		RequiresConfirmation:   false,
		MessengerRelatedUserID: &mu,
	}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(parent, nil)
	// becomes parent after update → needs tx
	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectCommit()

	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().UpdateTaskWithTx(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	taskRepo.EXPECT().GetChildTasksByParentID(gomock.Any(), taskID).Return([]*models.Task{}, nil)
	taskRepo.EXPECT().CreateTask(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, child *models.Task) (int64, error) {
		assert.NotNil(t, child.ParentID)
		assert.Equal(t, taskID, *child.ParentID)
		assert.Nil(t, child.CronExpression)
		return int64(2201), nil
	})
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil).AnyTimes()
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil).AnyTimes()
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	out, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{
		RequiresConfirmation: ptrBool(true),
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.RequiresConfirmation)
	assert.NotEmpty(t, pub.published)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_UpdateTask_ParentTitleChange_SyncsActiveChildren(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(30)
	mu := 7
	messengerID := int64(3)
	pub := &stubPublisher{}
	service.producer = pub

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectCommit()

	parent := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "old",
		Description:            "d",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(24 * time.Hour),
		CronExpression:         ptrString("0 9 * * *"),
		RequiresConfirmation:   true,
		MessengerRelatedUserID: &mu,
	}
	activeChild := &models.Task{
		ID: 301, Title: "old", Description: "d", UserID: 1, ParentID: &taskID,
		Status: string(models.TaskStatusScheduled), StartDate: time.Now().UTC().Add(time.Hour),
		MessengerRelatedUserID: &mu, RequiresConfirmation: true,
	}
	doneChild := &models.Task{ID: 302, Status: string(models.TaskStatusDone), ParentID: &taskID}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(parent, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().UpdateTaskWithTx(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).MinTimes(2)
	taskRepo.EXPECT().GetChildTasksByParentID(gomock.Any(), taskID).Return([]*models.Task{activeChild, doneChild}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil).AnyTimes()
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil).AnyTimes()
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	out, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{
		Title: ptrString("new title"),
	})
	require.NoError(t, err)
	assert.Equal(t, "new title", out.Title)
	assert.NotEmpty(t, pub.published)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_UpdateTask_ParentMute_PublishesDeletes(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(31)
	mu := 7
	messengerID := int64(3)
	pub := &stubPublisher{}
	service.producer = pub

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectCommit()

	parent := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "parent",
		Description:            "d",
		Status:                 string(models.TaskStatusScheduled),
		StartDate:              time.Now().UTC().Add(24 * time.Hour),
		CronExpression:         ptrString("0 9 * * *"),
		RequiresConfirmation:   true,
		MessengerRelatedUserID: &mu,
		Muted:                  false,
	}
	child := &models.Task{
		ID: 311, Title: "parent", Description: "d", UserID: 1, ParentID: &taskID,
		Status: string(models.TaskStatusScheduled), MessengerRelatedUserID: &mu, Muted: false,
	}

	taskRepo.EXPECT().GetTaskByID(gomock.Any(), taskID).Return(parent, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().UpdateTaskWithTx(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).MinTimes(2)
	taskRepo.EXPECT().GetChildTasksByParentID(gomock.Any(), taskID).Return([]*models.Task{child}, nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil).AnyTimes()
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil).AnyTimes()
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	out, err := service.UpdateTask(ctx, taskID, &models.TaskUpdateRequest{
		Muted: ptrBool(true),
	})
	require.NoError(t, err)
	assert.True(t, out.Muted)
	assert.NotEmpty(t, pub.published)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_DeleteTask_Transactional_Success(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(40)
	mu := 7
	messengerID := int64(3)
	pub := &stubPublisher{}
	service.producer = pub

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectCommit()

	task := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "single",
		Status:                 string(models.TaskStatusScheduled),
		MessengerRelatedUserID: &mu,
	}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(task, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().DeleteTaskWithTx(gomock.Any(), gomock.Any(), taskID).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	err = service.DeleteTask(ctx, taskID)
	require.NoError(t, err)
	require.Len(t, pub.published, 1)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_DeleteTask_ParentWithChildren_Success(t *testing.T) {
	service, taskRepo, _, messengerRepo, taskHistoryRepo, _ := setup(t)
	ctx := context.Background()
	taskID := int64(41)
	mu := 7
	messengerID := int64(3)
	pub := &stubPublisher{}
	service.producer = pub

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectCommit()

	task := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "parent",
		Status:                 string(models.TaskStatusScheduled),
		CronExpression:         ptrString("0 9 * * *"),
		RequiresConfirmation:   true,
		MessengerRelatedUserID: &mu,
	}
	children := []*models.Task{
		{ID: 411, ParentID: &taskID, Status: string(models.TaskStatusScheduled)},
		{ID: 412, ParentID: &taskID, Status: string(models.TaskStatusPending)},
	}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(task, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().GetChildTasksByParentID(gomock.Any(), taskID).Return(children, nil)
	taskRepo.EXPECT().DeleteChildTasksWithTx(gomock.Any(), gomock.Any(), taskID).Return(nil)
	taskRepo.EXPECT().DeleteTaskWithTx(gomock.Any(), gomock.Any(), taskID).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil).Times(2)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil).Times(2)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	err = service.DeleteTask(ctx, taskID)
	require.NoError(t, err)
	// 2 children + parent
	require.Len(t, pub.published, 3)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_DeleteTask_Parent_RollbackOnChildPublishError(t *testing.T) {
	service, taskRepo, _, messengerRepo, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(42)
	mu := 7
	messengerID := int64(3)
	service.producer = &stubPublisher{err: errors.New("amqp down")}

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectRollback()

	task := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "parent",
		Status:                 string(models.TaskStatusScheduled),
		RRule:                  ptrString("FREQ=DAILY;INTERVAL=1"),
		RequiresConfirmation:   true,
		MessengerRelatedUserID: &mu,
	}
	children := []*models.Task{{ID: 421, ParentID: &taskID, Status: string(models.TaskStatusScheduled)}}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(task, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().GetChildTasksByParentID(gomock.Any(), taskID).Return(children, nil)
	taskRepo.EXPECT().DeleteChildTasksWithTx(gomock.Any(), gomock.Any(), taskID).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)

	err = service.DeleteTask(ctx, taskID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to queue delete_task message for child task")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_DeleteTask_Transactional_RollbackOnPublishError(t *testing.T) {
	service, taskRepo, _, messengerRepo, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(43)
	mu := 7
	messengerID := int64(3)
	service.producer = &stubPublisher{err: errors.New("amqp down")}

	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mockDB.ExpectBegin()
	mockDB.ExpectRollback()

	task := &models.Task{
		ID:                     taskID,
		UserID:                 1,
		Title:                  "single",
		Status:                 string(models.TaskStatusScheduled),
		MessengerRelatedUserID: &mu,
	}

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(task, nil)
	taskRepo.EXPECT().GetDB().Return(sqlxDB)
	taskRepo.EXPECT().DeleteTaskWithTx(gomock.Any(), gomock.Any(), taskID).Return(nil)
	messengerRepo.EXPECT().GetMessengerRelatedUserByID(gomock.Any(), mu).Return(&models.MessengerRelatedUser{
		ID: int64(mu), MessengerID: &messengerID, ChatID: "c1",
	}, nil)
	messengerRepo.EXPECT().GetMessengerByID(gomock.Any(), messengerID).Return(&models.Messenger{ID: messengerID, Name: "telegram"}, nil)

	err = service.DeleteTask(ctx, taskID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to queue delete_task message")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestTaskService_DeleteTask_ParentWithoutDB_Fails(t *testing.T) {
	service, taskRepo, _, _, _, _ := setup(t)
	ctx := context.Background()
	taskID := int64(44)

	taskRepo.EXPECT().GetTaskByIDWithoutStatusFilter(gomock.Any(), taskID).Return(&models.Task{
		ID: taskID, CronExpression: ptrString("0 9 * * *"),
	}, nil)
	taskRepo.EXPECT().GetDB().Return(nil)

	err := service.DeleteTask(ctx, taskID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection not available for transactional delete of parent task")
}
