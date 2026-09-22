package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/googlecalendar"
)

func TestOutboxBackoff(t *testing.T) {
	assert.Equal(t, 1*time.Minute, OutboxBackoff(1))
	assert.Equal(t, 5*time.Minute, OutboxBackoff(2))
	assert.Equal(t, 30*time.Minute, OutboxBackoff(3))
	assert.Equal(t, 2*time.Hour, OutboxBackoff(4))
	assert.Equal(t, 2*time.Hour, OutboxBackoff(8))
	assert.Equal(t, 1*time.Minute, OutboxBackoff(0))
}

func TestMapEventToTaskFields(t *testing.T) {
	start := time.Now().UTC().Add(2 * time.Hour)
	end := start.Add(45 * time.Minute)
	ev := googlecalendar.Event{
		Summary:     " Meet ",
		Description: "desc",
		Start:       googlecalendar.EventDateTime{DateTime: &start},
		End:         googlecalendar.EventDateTime{DateTime: &end},
		Recurrence:  []string{"RRULE:FREQ=DAILY;COUNT=3"},
	}
	groupID := int64(9)
	task, err := MapEventToTaskFields(ev, 42, &groupID)
	require.NoError(t, err)
	assert.Equal(t, "Meet", task.Title)
	assert.Equal(t, "desc", task.Description)
	assert.Equal(t, int64(42), task.UserID)
	assert.Equal(t, &groupID, task.GroupID)
	assert.Equal(t, start.UTC(), task.StartDate)
	assert.Nil(t, task.FinishDate)
	require.NotNil(t, task.RRule)
	assert.Equal(t, "FREQ=DAILY;COUNT=3", *task.RRule)
	assert.Equal(t, string(models.TaskStatusScheduled), task.Status)
}

func TestMapEventToTaskFields_AllDay(t *testing.T) {
	ev := googlecalendar.Event{
		Summary: "",
		Start:   googlecalendar.EventDateTime{Date: "2026-09-21"},
		End:     googlecalendar.EventDateTime{Date: "2026-09-22"},
	}
	task, err := MapEventToTaskFields(ev, 1, nil)
	require.NoError(t, err)
	assert.Equal(t, "(untitled event)", task.Title)
	assert.Equal(t, time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC), task.StartDate)
}

func TestApplyDeletePolicy(t *testing.T) {
	task := &models.Task{ID: 1}
	soft, mute := ApplyDeletePolicy(models.CalendarDeletePolicySoftDeleteImported, task)
	assert.True(t, soft)
	assert.False(t, mute)

	soft, mute = ApplyDeletePolicy(models.CalendarDeletePolicyMuteImported, task)
	assert.False(t, soft)
	assert.True(t, mute)

	soft, mute = ApplyDeletePolicy(models.CalendarDeletePolicyKeep, task)
	assert.False(t, soft)
	assert.False(t, mute)
}

func TestShouldExportTask(t *testing.T) {
	parentID := int64(5)
	assert.True(t, ShouldExportTask(&models.Task{ID: 1}))
	assert.False(t, ShouldExportTask(&models.Task{ID: 2, ParentID: &parentID}))
	assert.False(t, ShouldExportTask(nil))
}

func TestBuildExportEvent(t *testing.T) {
	start := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	rrule := "FREQ=WEEKLY"
	task := &models.Task{ID: 77, Title: "T", Description: "D", StartDate: start, RRule: &rrule}
	ev := BuildExportEvent(task, 30, nil)
	require.NotNil(t, ev.Start.DateTime)
	require.NotNil(t, ev.End.DateTime)
	assert.Equal(t, start.Add(30*time.Minute), *ev.End.DateTime)
	assert.Equal(t, []string{"RRULE:FREQ=WEEKLY"}, ev.Recurrence)
	assert.Equal(t, "77", ev.ExtendedProperties[googlecalendar.PrivateExtendedPropertyTaskID])
}

func TestTaskIDFromExtendedProperties(t *testing.T) {
	id, ok := TaskIDFromExtendedProperties(map[string]string{googlecalendar.PrivateExtendedPropertyTaskID: "123"})
	assert.True(t, ok)
	assert.Equal(t, int64(123), id)
	_, ok = TaskIDFromExtendedProperties(nil)
	assert.False(t, ok)
}
