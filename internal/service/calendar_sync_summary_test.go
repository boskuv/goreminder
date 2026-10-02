package service

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/googlecalendar"
)

type stubCalendarClient struct {
	cal *googlecalendar.Calendar
	err error
}

func (s *stubCalendarClient) ListCalendars(context.Context) ([]googlecalendar.Calendar, error) {
	return nil, nil
}
func (s *stubCalendarClient) GetCalendar(context.Context, string) (*googlecalendar.Calendar, error) {
	return s.cal, s.err
}
func (s *stubCalendarClient) ListEvents(context.Context, string, googlecalendar.ListEventsOpts) (*googlecalendar.EventListResult, error) {
	return nil, nil
}
func (s *stubCalendarClient) GetEvent(context.Context, string, string) (*googlecalendar.Event, error) {
	return nil, nil
}
func (s *stubCalendarClient) CreateEvent(context.Context, string, *googlecalendar.Event) (*googlecalendar.Event, error) {
	return nil, nil
}
func (s *stubCalendarClient) PatchEvent(context.Context, string, string, *googlecalendar.Event) (*googlecalendar.Event, error) {
	return nil, nil
}
func (s *stubCalendarClient) DeleteEvent(context.Context, string, string) error { return nil }

func TestRefreshBindingCalendarSummary_UpdatesWhenChanged(t *testing.T) {
	old := "Old name"
	binding := &models.CalendarBinding{
		ID:               1,
		GoogleCalendarID: "cal-1",
		CalendarSummary:  &old,
	}
	repo := &stubBindingRepo{binding: binding}
	svc := &CalendarSyncService{
		bindings: repo,
		logger:   zerolog.Nop(),
	}
	client := &stubCalendarClient{cal: &googlecalendar.Calendar{ID: "cal-1", Summary: "New name"}}

	svc.refreshBindingCalendarSummary(context.Background(), client, binding)

	require.Equal(t, 1, repo.updateCalls)
	require.NotNil(t, binding.CalendarSummary)
	assert.Equal(t, "New name", *binding.CalendarSummary)
}

func TestRefreshBindingCalendarSummary_SkipsWhenUnchanged(t *testing.T) {
	name := "Same"
	binding := &models.CalendarBinding{
		ID:               1,
		GoogleCalendarID: "cal-1",
		CalendarSummary:  &name,
	}
	repo := &stubBindingRepo{binding: binding}
	svc := &CalendarSyncService{
		bindings: repo,
		logger:   zerolog.Nop(),
	}
	client := &stubCalendarClient{cal: &googlecalendar.Calendar{ID: "cal-1", Summary: "Same"}}

	svc.refreshBindingCalendarSummary(context.Background(), client, binding)

	assert.Equal(t, 0, repo.updateCalls)
}

func TestRefreshBindingCalendarSummary_IgnoresAPIError(t *testing.T) {
	name := "Keep"
	binding := &models.CalendarBinding{
		ID:               1,
		GoogleCalendarID: "cal-1",
		CalendarSummary:  &name,
	}
	repo := &stubBindingRepo{binding: binding}
	svc := &CalendarSyncService{
		bindings: repo,
		logger:   zerolog.Nop(),
	}
	client := &stubCalendarClient{err: assert.AnError}

	svc.refreshBindingCalendarSummary(context.Background(), client, binding)

	assert.Equal(t, 0, repo.updateCalls)
	assert.Equal(t, "Keep", *binding.CalendarSummary)
}

var _ googlecalendar.CalendarClient = (*stubCalendarClient)(nil)
