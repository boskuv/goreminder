package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskEvent_ToTaskMessage_ScheduleIncludesPreRemind(t *testing.T) {
	start := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	pre := int64(900)
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
	}
	msg := ev.ToTaskMessage()
	assert.Equal(t, "worker.schedule_task", msg.Task)
	require.Len(t, msg.Args, 9)
	assert.Equal(t, "telegram", msg.Args[0])
	assert.Equal(t, int64(42), msg.Args[2])
	assert.Equal(t, int64(900), msg.Args[8])
}

func TestTaskEvent_ToTaskMessage_ScheduleNilPreRemind(t *testing.T) {
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
	require.Len(t, msg.Args, 9)
	assert.Nil(t, msg.Args[8])
}
