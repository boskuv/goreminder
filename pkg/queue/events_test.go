package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskEvent_ToTaskMessage_ScheduleIncludesPreRemindAndRRule(t *testing.T) {
	start := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	pre := int64(900)
	rrule := "FREQ=MONTHLY;INTERVAL=3"
	ev := TaskEvent{
		Type:                   TaskEventSchedule,
		TaskID:                 42,
		MessengerName:          "telegram",
		ChatID:                 "123",
		Title:                  "Title",
		Description:            "Desc",
		StartDate:              &start,
		RequiresConfirmation:   true,
		PreRemindBeforeSeconds: &pre,
		RRule:                  &rrule,
	}
	msg := ev.ToTaskMessage()
	assert.Equal(t, "worker.schedule_task", msg.Task)
	require.Len(t, msg.Args, 10)
	assert.Equal(t, "telegram", msg.Args[0])
	assert.Equal(t, int64(42), msg.Args[2])
	assert.Nil(t, msg.Args[6])
	assert.Equal(t, true, msg.Args[7])
	assert.Equal(t, int64(900), msg.Args[8])
	assert.Equal(t, &rrule, msg.Args[9])
}

func TestTaskEvent_ToTaskMessage_ScheduleNilOptionalRecurrenceArgs(t *testing.T) {
	start := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	ev := TaskEvent{
		Type:          TaskEventSchedule,
		TaskID:        1,
		MessengerName: "telegram",
		ChatID:        "1",
		Title:         "t",
		StartDate:     &start,
	}
	msg := ev.ToTaskMessage()
	require.Len(t, msg.Args, 10)
	assert.Nil(t, msg.Args[6])
	assert.Nil(t, msg.Args[8])
	assert.Nil(t, msg.Args[9])
}

func TestTaskEvent_ToTaskMessage_ScheduleCronKeepsRRuleNil(t *testing.T) {
	start := time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
	cron := "0 9 * * *"
	ev := TaskEvent{
		Type:           TaskEventSchedule,
		TaskID:         7,
		MessengerName:  "telegram",
		ChatID:         "1",
		Title:          "t",
		StartDate:      &start,
		CronExpression: &cron,
	}
	msg := ev.ToTaskMessage()
	require.Len(t, msg.Args, 10)
	assert.Equal(t, &cron, msg.Args[6])
	assert.Nil(t, msg.Args[9])
}
