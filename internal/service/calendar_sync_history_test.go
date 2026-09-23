package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boskuv/goreminder/internal/models"
)

func TestCalendarTaskHistoryMap_IncludesSource(t *testing.T) {
	start := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	rrule := "FREQ=DAILY"
	mru := 3
	task := &models.Task{
		ID:                     42,
		Title:                  "Meet",
		UserID:                 7,
		StartDate:              start,
		Status:                 string(models.TaskStatusScheduled),
		RRule:                  &rrule,
		MessengerRelatedUserID: &mru,
	}
	m := calendarTaskHistoryMap(task)
	require.NotNil(t, m)
	assert.Equal(t, models.TaskHistorySourceGoogleCalendar, m[models.TaskHistorySourceKey])
	assert.Equal(t, int64(42), m["id"])
	assert.Equal(t, "Meet", m["title"])
	assert.Equal(t, "FREQ=DAILY", m["rrule"])
	assert.Equal(t, 3, m["messenger_related_user_id"])
}

func TestCalendarImportedFieldsChanged(t *testing.T) {
	base := &models.Task{
		Title:     "A",
		StartDate: time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
		Status:    string(models.TaskStatusScheduled),
	}
	same := *base
	assert.False(t, calendarImportedFieldsChanged(base, &same))

	other := *base
	other.Title = "B"
	assert.True(t, calendarImportedFieldsChanged(base, &other))

	done := *base
	done.Status = string(models.TaskStatusDone)
	assert.True(t, calendarImportedFieldsChanged(base, &done))
}
