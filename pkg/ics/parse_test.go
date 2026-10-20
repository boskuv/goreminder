package ics

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_OneShotFuture(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//GoReminder//TEST
BEGIN:VEVENT
UID:future-1@test
DTSTART:20990101T100000Z
DTEND:20990101T110000Z
SUMMARY:Future meeting
DESCRIPTION:Hello
END:VEVENT
END:VCALENDAR
`
	res, err := Parse(strings.NewReader(ics))
	require.NoError(t, err)
	require.Len(t, res.Events, 1)
	assert.Empty(t, res.Errors)
	assert.Equal(t, 0, res.SkippedTODO)
	ev := res.Events[0]
	assert.Equal(t, "future-1@test", ev.UID)
	assert.Equal(t, "Future meeting", ev.Summary)
	assert.Equal(t, "Hello", ev.Description)
	assert.False(t, ev.AllDay)
	assert.True(t, ev.Start.Equal(time.Date(2099, 1, 1, 10, 0, 0, 0, time.UTC)))
	require.NotNil(t, ev.End)
	assert.True(t, ev.End.Equal(time.Date(2099, 1, 1, 11, 0, 0, 0, time.UTC)))
	assert.Empty(t, ev.Recurrence)
}

func TestParse_RRuleAndAllDay(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:allday-rrule@test
DTSTART;VALUE=DATE:20200101
SUMMARY:Daily
RRULE:FREQ=DAILY;COUNT=5
END:VEVENT
END:VCALENDAR
`
	res, err := Parse(strings.NewReader(ics))
	require.NoError(t, err)
	require.Len(t, res.Events, 1)
	ev := res.Events[0]
	assert.True(t, ev.AllDay)
	assert.True(t, ev.Start.Equal(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)))
	require.Len(t, ev.Recurrence, 1)
	assert.Equal(t, "RRULE:FREQ=DAILY;COUNT=5", ev.Recurrence[0])
}

func TestParse_SkipsVTODO_PartialErrors(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTODO
UID:todo-1@test
SUMMARY:A todo
DUE:20990101T100000Z
END:VTODO
BEGIN:VEVENT
UID:bad@test
SUMMARY:No start
END:VEVENT
BEGIN:VEVENT
UID:ok@test
DTSTART:20990615T120000Z
SUMMARY:Ok event
END:VEVENT
END:VCALENDAR
`
	res, err := Parse(strings.NewReader(ics))
	require.NoError(t, err)
	assert.Equal(t, 1, res.SkippedTODO)
	require.Len(t, res.Errors, 1)
	assert.Equal(t, "bad@test", res.Errors[0].UID)
	require.Len(t, res.Events, 1)
	assert.Equal(t, "ok@test", res.Events[0].UID)
}

func TestParse_EmptySummary(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:empty-sum@test
DTSTART:20990101T090000Z
END:VEVENT
END:VCALENDAR
`
	res, err := Parse(strings.NewReader(ics))
	require.NoError(t, err)
	require.Len(t, res.Events, 1)
	assert.Equal(t, "", res.Events[0].Summary)
}

func TestParse_InvalidCalendar(t *testing.T) {
	_, err := Parse(strings.NewReader("not an ics"))
	require.Error(t, err)
}

func TestParse_TestdataSample(t *testing.T) {
	f, err := os.Open("testdata/sample.ics")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	res, err := Parse(f)
	require.NoError(t, err)
	require.Len(t, res.Events, 2)
	assert.Equal(t, "Sample future event", res.Events[0].Summary)
	require.Len(t, res.Events[1].Recurrence, 1)
	assert.Contains(t, res.Events[1].Recurrence[0], "FREQ=WEEKLY")
}
