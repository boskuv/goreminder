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

func TestBindingSyncBackoff(t *testing.T) {
	assert.Equal(t, OutboxBackoff(1), BindingSyncBackoff(1))
	assert.Equal(t, OutboxBackoff(3), BindingSyncBackoff(3))
	assert.Equal(t, MaxBindingSyncAttempts, 8)
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
	mru := 3
	task, err := MapEventToTaskFields(ev, 42, &groupID, &mru)
	require.NoError(t, err)
	assert.Equal(t, "Meet", task.Title)
	assert.Equal(t, "desc", task.Description)
	assert.Equal(t, int64(42), task.UserID)
	assert.Equal(t, &groupID, task.GroupID)
	assert.Equal(t, &mru, task.MessengerRelatedUserID)
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
	task, err := MapEventToTaskFields(ev, 1, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "(untitled event)", task.Title)
	assert.Equal(t, time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC), task.StartDate)
	assert.Equal(t, string(models.TaskStatusDone), task.Status)
	require.NotNil(t, task.FinishDate)
}

func TestMapEventToTaskFields_RecurringPastAdvances(t *testing.T) {
	// Past DTSTART with daily RRULE → start_date should jump to a future occurrence.
	start := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
	ev := googlecalendar.Event{
		Summary:    "Daily stand-up",
		Start:      googlecalendar.EventDateTime{DateTime: &start},
		Recurrence: []string{"RRULE:FREQ=DAILY"},
	}
	task, err := MapEventToTaskFields(ev, 1, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, task.RRule)
	assert.True(t, task.StartDate.After(time.Now().UTC()), "expected next occurrence in the future, got %s", task.StartDate)
	assert.Equal(t, string(models.TaskStatusScheduled), task.Status)
	assert.Nil(t, task.FinishDate)
}

func TestMapEventToTaskFields_PastOneShotDone(t *testing.T) {
	start := time.Now().UTC().Add(-2 * time.Hour)
	end := start.Add(30 * time.Minute)
	ev := googlecalendar.Event{
		Summary: "Past meeting",
		Start:   googlecalendar.EventDateTime{DateTime: &start},
		End:     googlecalendar.EventDateTime{DateTime: &end},
	}
	task, err := MapEventToTaskFields(ev, 1, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, string(models.TaskStatusDone), task.Status)
	require.NotNil(t, task.FinishDate)
	assert.Equal(t, end.UTC(), task.FinishDate.UTC())
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

func TestBindingAllowsTaskExport(t *testing.T) {
	assert.False(t, bindingAllowsTaskExport(models.CalendarBindingDirectionImport))
	assert.True(t, bindingAllowsTaskExport(models.CalendarBindingDirectionExport))
	assert.True(t, bindingAllowsTaskExport(models.CalendarBindingDirectionBoth))
	assert.False(t, bindingAllowsTaskExport(""))
}

func TestBuildExportEvent(t *testing.T) {
	start := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	rrule := "FREQ=WEEKLY"
	task := &models.Task{ID: 77, Title: "T", Description: "D", StartDate: start, RRule: &rrule}
	ev := BuildExportEvent(task, 30, nil)
	require.NotNil(t, ev.Start.DateTime)
	require.NotNil(t, ev.End.DateTime)
	assert.Equal(t, start.Add(30*time.Minute), *ev.End.DateTime)
	assert.Equal(t, "UTC", ev.Start.TimeZone)
	assert.Equal(t, "UTC", ev.End.TimeZone)
	assert.Equal(t, []string{"RRULE:FREQ=WEEKLY"}, ev.Recurrence)
	assert.Equal(t, "77", ev.ExtendedProperties[googlecalendar.PrivateExtendedPropertyTaskID])
}

func TestBuildExportEvent_CronMapsToRRule(t *testing.T) {
	start := time.Date(2026, 9, 21, 9, 0, 0, 0, time.UTC)
	cronExpr := "0 9 * * *"
	task := &models.Task{ID: 1, Title: "Daily", StartDate: start, CronExpression: &cronExpr}
	ev := BuildExportEvent(task, 30, nil)
	assert.Equal(t, []string{"RRULE:FREQ=DAILY"}, ev.Recurrence)
}

func TestCronExpressionToRRule(t *testing.T) {
	cases := []struct {
		cron string
		want string
	}{
		{"0 9 * * *", "FREQ=DAILY"},
		{"30 8 * * 1", "FREQ=WEEKLY;BYDAY=MO"},
		{"0 9 * * 1,3,5", "FREQ=WEEKLY;BYDAY=MO,WE,FR"},
		{"0 9 * * 1-5", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"},
		{"0 12 15 * *", "FREQ=MONTHLY;BYMONTHDAY=15"},
		{"0 12 1 1 *", "FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1"},
	}
	for _, tc := range cases {
		t.Run(tc.cron, func(t *testing.T) {
			got := CronExpressionToRRule(tc.cron)
			require.NotNil(t, got)
			assert.Equal(t, tc.want, *got)
		})
	}
	assert.Nil(t, CronExpressionToRRule("*/5 * * * *"))
	assert.Nil(t, CronExpressionToRRule("0 9 1 * 1")) // DOM+DOW combined
	assert.Nil(t, CronExpressionToRRule(""))
}

func TestTaskIDFromExtendedProperties(t *testing.T) {
	id, ok := TaskIDFromExtendedProperties(map[string]string{googlecalendar.PrivateExtendedPropertyTaskID: "123"})
	assert.True(t, ok)
	assert.Equal(t, int64(123), id)
	_, ok = TaskIDFromExtendedProperties(nil)
	assert.False(t, ok)
}
