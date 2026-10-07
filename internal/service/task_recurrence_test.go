package service

import (
	"testing"
	"time"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/stretchr/testify/require"
)

func TestNextStartFromRRule_Daily(t *testing.T) {
	now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	seriesStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	next, err := nextStartFromRRule(now, "FREQ=DAILY;INTERVAL=1", seriesStart)
	require.NoError(t, err)
	require.True(t, next.After(now))
}

func TestValidateCronExpressionAndRRuleExclusive(t *testing.T) {
	require.NoError(t, validateCronExpressionAndRRuleExclusive(nil, nil))
	require.NoError(t, validateCronExpressionAndRRuleExclusive(ptrString("0 * * * *"), nil))
	require.NoError(t, validateCronExpressionAndRRuleExclusive(nil, ptrString("FREQ=DAILY")))
	require.Error(t, validateCronExpressionAndRRuleExclusive(ptrString("0 * * * *"), ptrString("FREQ=DAILY")))
}

func TestNextExecutableStartDate_CronPast(t *testing.T) {
	cron := "0 9 * * *"
	past := time.Date(2020, 1, 1, 9, 0, 0, 0, time.UTC)
	now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	task := &models.Task{
		StartDate:            past,
		CronExpression:       &cron,
		RequiresConfirmation: false,
	}
	next, err := nextExecutableStartDate(task, nil, now)
	require.NoError(t, err)
	require.True(t, next.After(now))
}

func TestNextExecutableStartDate_OneTimePastReturnsZero(t *testing.T) {
	past := time.Date(2020, 1, 1, 9, 0, 0, 0, time.UTC)
	now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	task := &models.Task{StartDate: past}
	next, err := nextExecutableStartDate(task, nil, now)
	require.NoError(t, err)
	require.True(t, next.IsZero())
}

func TestShouldRepublishScheduleAfterUnmute_ParentWithConfirmation(t *testing.T) {
	cron := "0 9 * * *"
	parent := &models.Task{
		CronExpression:       &cron,
		RequiresConfirmation: true,
	}
	require.False(t, shouldRepublishScheduleAfterUnmute(parent))
}

func TestShiftRecurrenceFromCompletion_WeeklyCron(t *testing.T) {
	cron := "30 8 * * 1"
	parent := &models.Task{
		StartDate:      time.Date(2026, 9, 7, 8, 30, 0, 0, time.UTC),
		CronExpression: &cron,
	}
	clock := time.Date(2026, 10, 5, 8, 30, 0, 0, time.UTC)      // Monday
	completed := time.Date(2026, 10, 7, 15, 30, 0, 0, time.UTC) // Wednesday

	got, err := shiftRecurrenceFromCompletion(parent, clock, completed)
	require.NoError(t, err)
	require.NotNil(t, got.Cron)
	require.Nil(t, got.RRule)
	require.Equal(t, "30 8 * * 3", *got.Cron)
	require.True(t, got.Anchor.Equal(time.Date(2026, 10, 7, 8, 30, 0, 0, time.UTC)))
	require.True(t, got.NextStart.Equal(time.Date(2026, 10, 14, 8, 30, 0, 0, time.UTC)))
	require.Equal(t, cron, *parent.CronExpression, "parent rule must not be mutated in place")
}

func TestShiftRecurrenceFromCompletion_MonthlyRRule(t *testing.T) {
	rrule := "FREQ=MONTHLY;BYMONTHDAY=1"
	parent := &models.Task{
		StartDate: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
		RRule:     &rrule,
	}
	clock := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 10, 7, 15, 0, 0, 0, time.UTC)

	got, err := shiftRecurrenceFromCompletion(parent, clock, completed)
	require.NoError(t, err)
	require.NotNil(t, got.RRule)
	require.Equal(t, "FREQ=MONTHLY;BYMONTHDAY=7", *got.RRule)
	require.True(t, got.Anchor.Equal(time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)))
	require.True(t, got.NextStart.Equal(time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC)))
}

func TestShiftRecurrenceFromCompletion_DailyRRuleKeepsString(t *testing.T) {
	rrule := "FREQ=DAILY;INTERVAL=1"
	parent := &models.Task{
		StartDate: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
		RRule:     &rrule,
	}
	clock := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 10, 7, 15, 0, 0, 0, time.UTC)

	got, err := shiftRecurrenceFromCompletion(parent, clock, completed)
	require.NoError(t, err)
	require.NotNil(t, got.RRule)
	require.Equal(t, rrule, *got.RRule)
	require.True(t, got.Anchor.Equal(time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)))
	require.True(t, got.NextStart.Equal(time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)))
}

func TestShiftRecurrenceFromCompletion_YearlyCron(t *testing.T) {
	cron := "0 12 1 1 *"
	parent := &models.Task{CronExpression: &cron}
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 3, 7, 18, 0, 0, 0, time.UTC)

	got, err := shiftRecurrenceFromCompletion(parent, clock, completed)
	require.NoError(t, err)
	require.Equal(t, "0 12 7 3 *", *got.Cron)
	require.True(t, got.NextStart.Equal(time.Date(2027, 3, 7, 12, 0, 0, 0, time.UTC)))
}

func TestShiftRecurrenceFromCompletion_RejectsComplexRules(t *testing.T) {
	cases := []struct {
		name string
		cron *string
		rule *string
	}{
		{name: "cron weekday list", cron: ptrString("0 9 * * 1,3,5")},
		{name: "cron step", cron: ptrString("0 9 */2 * *")},
		{name: "rrule byday list", rule: ptrString("FREQ=WEEKLY;BYDAY=MO,WE")},
		{name: "rrule count", rule: ptrString("FREQ=DAILY;COUNT=5")},
		{name: "rrule hourly", rule: ptrString("FREQ=HOURLY;INTERVAL=1")},
		{name: "rrule nth weekday", rule: ptrString("FREQ=MONTHLY;BYDAY=1MO")},
	}
	clock := time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parent := &models.Task{CronExpression: tc.cron, RRule: tc.rule, StartDate: clock}
			_, err := shiftRecurrenceFromCompletion(parent, clock, clock.Add(time.Hour))
			require.ErrorIs(t, err, errRecurrenceNotShiftable)
		})
	}
}

func TestRecurrenceFieldChanged(t *testing.T) {
	oldCron := ptrString("0 9 * * *")

	require.False(t, recurrenceFieldChanged(nil, oldCron), "missing field in partial update must not be treated as changed")
	require.False(t, recurrenceFieldChanged(ptrString("0 9 * * *"), oldCron), "same value must not be treated as changed")
	require.True(t, recurrenceFieldChanged(ptrString("0 10 * * *"), oldCron), "different value must be treated as changed")
	require.False(t, recurrenceFieldChanged(ptrString(""), nil), "empty string and nil should be treated as equivalent unset values")
	require.True(t, recurrenceFieldChanged(ptrString("FREQ=DAILY"), nil), "setting previously unset recurrence must be treated as changed")
}
