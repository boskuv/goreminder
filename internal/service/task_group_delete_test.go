package service

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
)

type stubTaskGroupRepo struct {
	group       *models.TaskGroup
	getErr      error
	deleted     bool
	deleteCalls int
}

func (s *stubTaskGroupRepo) CreateTaskGroup(context.Context, *models.TaskGroup) (int64, error) {
	return 0, nil
}
func (s *stubTaskGroupRepo) GetTaskGroupByID(context.Context, int64) (*models.TaskGroup, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.group, nil
}
func (s *stubTaskGroupRepo) GetAllTaskGroups(context.Context, int, int, string, *int64) ([]*models.TaskGroup, int, error) {
	return nil, 0, nil
}
func (s *stubTaskGroupRepo) UpdateTaskGroup(context.Context, *models.TaskGroup) error { return nil }
func (s *stubTaskGroupRepo) DeleteTaskGroup(context.Context, int64) error {
	s.deleteCalls++
	s.deleted = true
	return nil
}

type countBindingsStub struct {
	stubBindingRepo
	count int
}

func (s *countBindingsStub) CountByGroupID(context.Context, int64) (int, error) {
	return s.count, nil
}

func TestDeleteTaskGroup_BlockedWhenCalendarBindingExists(t *testing.T) {
	groups := &stubTaskGroupRepo{group: &models.TaskGroup{ID: 10, UserID: 1, Name: "Export"}}
	svc := NewTaskGroupService(groups, nil, nil, zerolog.Nop(), &countBindingsStub{count: 2})

	err := svc.DeleteTaskGroup(context.Background(), 10)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConflict))
	assert.Equal(t, 0, groups.deleteCalls)
}

func TestDeleteTaskGroup_AllowedWhenNoBindings(t *testing.T) {
	groups := &stubTaskGroupRepo{group: &models.TaskGroup{
		ID: 11, UserID: 1, Name: "Tmp", CreatedAt: time.Now().UTC(),
	}}
	svc := NewTaskGroupService(groups, nil, nil, zerolog.Nop(), &countBindingsStub{count: 0})

	err := svc.DeleteTaskGroup(context.Background(), 11)
	require.NoError(t, err)
	assert.Equal(t, 1, groups.deleteCalls)
}
