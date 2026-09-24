package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	errs "github.com/boskuv/goreminder/internal/errors"
	mock_repository "github.com/boskuv/goreminder/internal/mocks/repository"
	"github.com/boskuv/goreminder/internal/models"
)

type stubSyncLinkRepo struct {
	link *models.TaskSyncLink
	err  error
}

func (s *stubSyncLinkRepo) Create(context.Context, *models.TaskSyncLink) (int64, error) {
	return 0, nil
}
func (s *stubSyncLinkRepo) GetByTaskID(context.Context, int64) (*models.TaskSyncLink, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.link, nil
}
func (s *stubSyncLinkRepo) GetByEvent(context.Context, string, string, string) (*models.TaskSyncLink, error) {
	return nil, errs.ErrNotFound
}
func (s *stubSyncLinkRepo) ListByBindingID(context.Context, int64) ([]*models.TaskSyncLink, error) {
	return nil, nil
}
func (s *stubSyncLinkRepo) ListByCalendarIDAndOrigin(context.Context, string, models.TaskSyncLinkOrigin) ([]*models.TaskSyncLink, error) {
	return nil, nil
}
func (s *stubSyncLinkRepo) ListTaskIDsByProvider(context.Context, int64, string) ([]int64, error) {
	return nil, nil
}
func (s *stubSyncLinkRepo) Update(context.Context, *models.TaskSyncLink) error { return nil }
func (s *stubSyncLinkRepo) Delete(context.Context, int64) error                 { return nil }
func (s *stubSyncLinkRepo) SetSyncEnabled(context.Context, int64, bool) error  { return nil }

type stubBindingRepo struct {
	binding  *models.CalendarBinding
	bindings []*models.CalendarBinding
}

func (s *stubBindingRepo) Create(context.Context, *models.CalendarBinding) (int64, error) {
	return 0, nil
}
func (s *stubBindingRepo) GetByID(_ context.Context, id int64) (*models.CalendarBinding, error) {
	if s.binding != nil && s.binding.ID == id {
		return s.binding, nil
	}
	for _, b := range s.bindings {
		if b.ID == id {
			return b, nil
		}
	}
	if s.binding != nil {
		return s.binding, nil
	}
	return nil, errs.ErrNotFound
}
func (s *stubBindingRepo) ListByUserID(context.Context, int64) ([]*models.CalendarBinding, error) {
	if s.bindings != nil {
		return s.bindings, nil
	}
	if s.binding != nil {
		return []*models.CalendarBinding{s.binding}, nil
	}
	return nil, nil
}
func (s *stubBindingRepo) Update(context.Context, *models.CalendarBinding) error { return nil }
func (s *stubBindingRepo) SoftDelete(context.Context, int64) error               { return nil }
func (s *stubBindingRepo) ListActiveForSync(context.Context) ([]*models.CalendarBinding, error) {
	return nil, nil
}
func (s *stubBindingRepo) ListDueForSync(context.Context, time.Time, int) ([]*models.CalendarBinding, error) {
	return nil, nil
}
func (s *stubBindingRepo) CountByGroupID(context.Context, int64) (int, error) { return 0, nil }
func (s *stubBindingRepo) ClearSyncToken(context.Context, int64) error { return nil }

type stubOutboxRepo struct {
	kinds []string
}

func (s *stubOutboxRepo) Enqueue(_ context.Context, kind string, _ json.RawMessage) (int64, error) {
	s.kinds = append(s.kinds, kind)
	return int64(len(s.kinds)), nil
}
func (s *stubOutboxRepo) ClaimDue(context.Context, int) ([]*models.SyncOutbox, error) {
	return nil, nil
}
func (s *stubOutboxRepo) MarkDone(context.Context, int64) error { return nil }
func (s *stubOutboxRepo) MarkRetry(context.Context, int64, int, time.Time, string) error {
	return nil
}
func (s *stubOutboxRepo) MarkFailed(context.Context, int64, string) error { return nil }
func (s *stubOutboxRepo) CountPending(context.Context) (int, error)       { return 0, nil }
func (s *stubOutboxRepo) CountByUserID(context.Context, int64) (int, int, int, error) {
	return 0, 0, 0, nil
}

func TestEnqueueExport_DeletedAfterSoftDelete_UsesSyncLink(t *testing.T) {
	ctrl := gomock.NewController(t)
	tasks := mock_repository.NewMockTaskRepository(ctrl)
	tasks.EXPECT().
		GetTaskByIDWithoutStatusFilter(gomock.Any(), int64(1585)).
		Return(nil, errs.ErrNotFound)

	bindingID := int64(7)
	outbox := &stubOutboxRepo{}
	svc := &CalendarSyncService{
		tasks:     tasks,
		syncLinks: &stubSyncLinkRepo{link: &models.TaskSyncLink{
			ID:                11,
			TaskID:            1585,
			Origin:            models.TaskSyncLinkOriginExported,
			SyncEnabled:       true,
			CalendarBindingID: &bindingID,
			GoogleEventID:     "evt-1",
			GoogleCalendarID:  "cal-1",
		}},
		bindings: &stubBindingRepo{binding: &models.CalendarBinding{
			ID:        bindingID,
			Direction: models.CalendarBindingDirectionExport,
		}},
		outbox: outbox,
		logger: zerolog.Nop(),
	}

	err := svc.EnqueueExport(context.Background(), 1585, "deleted")
	require.NoError(t, err)
	require.Len(t, outbox.kinds, 1)
	assert.Equal(t, OutboxKindExportDelete, outbox.kinds[0])
}

func TestEnqueueExport_DeletedAfterSoftDelete_ImportOnlyNoEnqueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	tasks := mock_repository.NewMockTaskRepository(ctrl)
	tasks.EXPECT().
		GetTaskByIDWithoutStatusFilter(gomock.Any(), int64(99)).
		Return(nil, errs.ErrNotFound)

	bindingID := int64(3)
	outbox := &stubOutboxRepo{}
	svc := &CalendarSyncService{
		tasks: tasks,
		syncLinks: &stubSyncLinkRepo{link: &models.TaskSyncLink{
			ID:                5,
			TaskID:            99,
			Origin:            models.TaskSyncLinkOriginImported,
			SyncEnabled:       true,
			CalendarBindingID: &bindingID,
		}},
		bindings: &stubBindingRepo{binding: &models.CalendarBinding{
			ID:        bindingID,
			Direction: models.CalendarBindingDirectionImport,
		}},
		outbox: outbox,
		logger: zerolog.Nop(),
	}

	err := svc.EnqueueExport(context.Background(), 99, "deleted")
	require.NoError(t, err)
	assert.Empty(t, outbox.kinds)
}

func TestEnqueueExport_LeaveGroupWithoutOptIn_EnqueuesDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	tasks := mock_repository.NewMockTaskRepository(ctrl)
	exportGroup := int64(3)
	bindingID := int64(7)
	tasks.EXPECT().
		GetTaskByIDWithoutStatusFilter(gomock.Any(), int64(42)).
		Return(&models.Task{ID: 42, UserID: 1, GroupID: nil}, nil) // left group

	binding := &models.CalendarBinding{
		ID:        bindingID,
		UserID:    1,
		Direction: models.CalendarBindingDirectionExport,
		GroupID:   &exportGroup,
	}
	outbox := &stubOutboxRepo{}
	svc := &CalendarSyncService{
		tasks: tasks,
		syncLinks: &stubSyncLinkRepo{link: &models.TaskSyncLink{
			TaskID:            42,
			Origin:            models.TaskSyncLinkOriginExported,
			SyncEnabled:       true,
			ExportOptIn:       false,
			CalendarBindingID: &bindingID,
		}},
		bindings: &stubBindingRepo{binding: binding, bindings: []*models.CalendarBinding{binding}},
		outbox:   outbox,
		logger:   zerolog.Nop(),
	}

	err := svc.EnqueueExport(context.Background(), 42, "updated")
	require.NoError(t, err)
	require.Len(t, outbox.kinds, 1)
	assert.Equal(t, OutboxKindExportDelete, outbox.kinds[0])
}

func TestEnqueueExport_LeaveGroupWithOptIn_StillUpserts(t *testing.T) {
	ctrl := gomock.NewController(t)
	tasks := mock_repository.NewMockTaskRepository(ctrl)
	exportGroup := int64(3)
	bindingID := int64(7)
	tasks.EXPECT().
		GetTaskByIDWithoutStatusFilter(gomock.Any(), int64(43)).
		Return(&models.Task{ID: 43, UserID: 1, GroupID: nil}, nil)

	binding := &models.CalendarBinding{
		ID:        bindingID,
		UserID:    1,
		Direction: models.CalendarBindingDirectionExport,
		GroupID:   &exportGroup,
	}
	outbox := &stubOutboxRepo{}
	svc := &CalendarSyncService{
		tasks: tasks,
		syncLinks: &stubSyncLinkRepo{link: &models.TaskSyncLink{
			TaskID:            43,
			Origin:            models.TaskSyncLinkOriginExported,
			SyncEnabled:       true,
			ExportOptIn:       true,
			CalendarBindingID: &bindingID,
		}},
		bindings: &stubBindingRepo{binding: binding, bindings: []*models.CalendarBinding{binding}},
		outbox:   outbox,
		logger:   zerolog.Nop(),
	}

	err := svc.EnqueueExport(context.Background(), 43, "updated")
	require.NoError(t, err)
	require.Len(t, outbox.kinds, 1)
	assert.Equal(t, OutboxKindExportUpsert, outbox.kinds[0])
}
