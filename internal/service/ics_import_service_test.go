package service

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	errs "github.com/boskuv/goreminder/internal/errors"
	mock_repositories "github.com/boskuv/goreminder/internal/mocks/repository"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/attachments"
	"github.com/boskuv/goreminder/pkg/ics"
	"github.com/boskuv/goreminder/pkg/logger"
)

type stubICSTaskGroupRepo struct {
	group *models.TaskGroup
	err   error
}

func (s *stubICSTaskGroupRepo) CreateTaskGroup(context.Context, *models.TaskGroup) (int64, error) {
	panic("unexpected")
}
func (s *stubICSTaskGroupRepo) GetTaskGroupByID(_ context.Context, id int64) (*models.TaskGroup, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.group == nil || s.group.ID != id {
		return nil, errs.ErrNotFound
	}
	return s.group, nil
}
func (s *stubICSTaskGroupRepo) GetAllTaskGroups(context.Context, int, int, string, *int64) ([]*models.TaskGroup, int, error) {
	panic("unexpected")
}
func (s *stubICSTaskGroupRepo) UpdateTaskGroup(context.Context, *models.TaskGroup) error {
	panic("unexpected")
}
func (s *stubICSTaskGroupRepo) DeleteTaskGroup(context.Context, int64) error {
	panic("unexpected")
}

func TestICSEventToGoogle_AllDayAndRRule(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	ev := ics.Event{
		UID:        "u1",
		Summary:    "Daily",
		AllDay:     true,
		Start:      start,
		End:        &end,
		Recurrence: []string{"RRULE:FREQ=DAILY"},
	}
	gc := icsEventToGoogle(ev)
	assert.Equal(t, "2020-01-01", gc.Start.Date)
	assert.Equal(t, "2020-01-02", gc.End.Date)
	assert.Nil(t, gc.Start.DateTime)
	require.Len(t, gc.Recurrence, 1)
}

func TestICSImportService_Import_CreatesWithoutConfirmation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	taskRepo := mock_repositories.NewMockTaskRepository(ctrl)
	userRepo := mock_repositories.NewMockUserRepository(ctrl)
	messengerRepo := mock_repositories.NewMockMessengerRepository(ctrl)
	taskHistoryRepo := mock_repositories.NewMockTaskHistoryRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	pub := &stubPublisher{}

	taskService := NewTaskService(taskRepo, userRepo, messengerRepo, taskHistoryRepo, pub, attachments.NewNoopClient(), NoopActivityTracker{}, false, testLogger)
	groups := &stubICSTaskGroupRepo{}
	icsSvc := NewICSImportService(taskService, userRepo, groups, messengerRepo, testLogger)

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(7)).Return(&models.User{ID: 7}, nil).Times(2) // import check + CreateTask
	taskRepo.EXPECT().CreateTask(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, task *models.Task) (int64, error) {
		assert.False(t, task.RequiresConfirmation)
		assert.Equal(t, models.TaskHistorySourceICSImport, task.HistorySource)
		assert.Equal(t, "ICS meeting", task.Title)
		assert.Nil(t, task.ParentID)
		assert.Equal(t, string(models.TaskStatusScheduled), task.Status)
		return 101, nil
	})
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, h *models.TaskHistory) error {
		require.NotNil(t, h.NewValue)
		assert.Equal(t, models.TaskHistorySourceICSImport, h.NewValue[models.TaskHistorySourceKey])
		return nil
	})

	body := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:ics-1@test
DTSTART:20990101T150000Z
SUMMARY:ICS meeting
END:VEVENT
END:VCALENDAR
`
	res, err := icsSvc.Import(context.Background(), strings.NewReader(body), ICSImportOptions{UserID: 7})
	require.NoError(t, err)
	require.Equal(t, []int64{101}, res.CreatedTaskIDs)
	assert.Empty(t, res.Errors)
	assert.Equal(t, 0, res.SkippedTODO)
}

func TestICSImportService_Import_PartialErrorsAndVTODO(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	taskRepo := mock_repositories.NewMockTaskRepository(ctrl)
	userRepo := mock_repositories.NewMockUserRepository(ctrl)
	messengerRepo := mock_repositories.NewMockMessengerRepository(ctrl)
	taskHistoryRepo := mock_repositories.NewMockTaskHistoryRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)

	taskService := NewTaskService(taskRepo, userRepo, messengerRepo, taskHistoryRepo, &stubPublisher{}, attachments.NewNoopClient(), NoopActivityTracker{}, false, testLogger)
	icsSvc := NewICSImportService(taskService, userRepo, &stubICSTaskGroupRepo{}, messengerRepo, testLogger)

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(&models.User{ID: 1}, nil).AnyTimes()
	taskRepo.EXPECT().CreateTask(gomock.Any(), gomock.Any()).Return(int64(55), nil)
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	body := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTODO
UID:todo@test
SUMMARY:Skip me
END:VTODO
BEGIN:VEVENT
UID:bad@test
SUMMARY:No DTSTART
END:VEVENT
BEGIN:VEVENT
UID:ok@test
DTSTART:20990601T080000Z
SUMMARY:Ok
END:VEVENT
END:VCALENDAR
`
	res, err := icsSvc.Import(context.Background(), strings.NewReader(body), ICSImportOptions{UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, res.SkippedTODO)
	assert.Equal(t, []int64{55}, res.CreatedTaskIDs)
	require.NotEmpty(t, res.Errors)
	assert.Equal(t, "bad@test", res.Errors[0].UID)
}

func TestICSImportService_Import_PastOneShotDone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	taskRepo := mock_repositories.NewMockTaskRepository(ctrl)
	userRepo := mock_repositories.NewMockUserRepository(ctrl)
	messengerRepo := mock_repositories.NewMockMessengerRepository(ctrl)
	taskHistoryRepo := mock_repositories.NewMockTaskHistoryRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)

	taskService := NewTaskService(taskRepo, userRepo, messengerRepo, taskHistoryRepo, &stubPublisher{}, attachments.NewNoopClient(), NoopActivityTracker{}, false, testLogger)
	icsSvc := NewICSImportService(taskService, userRepo, &stubICSTaskGroupRepo{}, messengerRepo, testLogger)

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(2)).Return(&models.User{ID: 2}, nil).AnyTimes()
	taskRepo.EXPECT().CreateTask(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, task *models.Task) (int64, error) {
		assert.Equal(t, string(models.TaskStatusDone), task.Status)
		require.NotNil(t, task.FinishDate)
		return 9, nil
	})
	taskHistoryRepo.EXPECT().CreateTaskHistory(gomock.Any(), gomock.Any()).Return(nil)

	body := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:past@test
DTSTART:20200101T100000Z
SUMMARY:Past
END:VEVENT
END:VCALENDAR
`
	res, err := icsSvc.Import(context.Background(), strings.NewReader(body), ICSImportOptions{UserID: 2})
	require.NoError(t, err)
	assert.Equal(t, []int64{9}, res.CreatedTaskIDs)
}

func TestICSImportService_Import_InvalidUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := mock_repositories.NewMockUserRepository(ctrl)
	messengerRepo := mock_repositories.NewMockMessengerRepository(ctrl)
	taskRepo := mock_repositories.NewMockTaskRepository(ctrl)
	taskHistoryRepo := mock_repositories.NewMockTaskHistoryRepository(ctrl)
	testLogger := logger.New(io.Discard, zerolog.DebugLevel, false)
	taskService := NewTaskService(taskRepo, userRepo, messengerRepo, taskHistoryRepo, &stubPublisher{}, attachments.NewNoopClient(), NoopActivityTracker{}, false, testLogger)
	icsSvc := NewICSImportService(taskService, userRepo, &stubICSTaskGroupRepo{}, messengerRepo, testLogger)

	userRepo.EXPECT().GetUserByID(gomock.Any(), int64(99)).Return(nil, errs.ErrNotFound)
	_, err := icsSvc.Import(context.Background(), strings.NewReader("BEGIN:VCALENDAR\nEND:VCALENDAR\n"), ICSImportOptions{UserID: 99})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrUnprocessableEntity))
}
