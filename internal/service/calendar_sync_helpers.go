package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/googlecalendar"
)

const (
	OutboxKindExportUpsert = "export_upsert"
	OutboxKindExportDelete = "export_delete"

	maxOutboxAttempts = 8
)

var outboxBackoffSteps = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
}

// OutboxBackoff returns the delay before the next retry for the given 1-based attempt count.
func OutboxBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return outboxBackoffSteps[0]
	}
	idx := attempts - 1
	if idx >= len(outboxBackoffSteps) {
		return outboxBackoffSteps[len(outboxBackoffSteps)-1]
	}
	return outboxBackoffSteps[idx]
}

// EventStartTime extracts the start instant from a Google event.
func EventStartTime(ev googlecalendar.Event) (time.Time, error) {
	if ev.Start.DateTime != nil {
		return ev.Start.DateTime.UTC(), nil
	}
	if ev.Start.Date != "" {
		t, err := time.ParseInLocation("2006-01-02", ev.Start.Date, time.UTC)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse all-day start: %w", err)
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("event has no start")
}

// EventEndTime extracts the end instant from a Google event.
func EventEndTime(ev googlecalendar.Event) *time.Time {
	if ev.End.DateTime != nil {
		t := ev.End.DateTime.UTC()
		return &t
	}
	if ev.End.Date != "" {
		t, err := time.ParseInLocation("2006-01-02", ev.End.Date, time.UTC)
		if err != nil {
			return nil
		}
		return &t
	}
	return nil
}

// EventDurationSeconds returns duration between start and end when both exist.
func EventDurationSeconds(ev googlecalendar.Event) *int {
	start, err := EventStartTime(ev)
	if err != nil {
		return nil
	}
	end := EventEndTime(ev)
	if end == nil {
		return nil
	}
	secs := int(end.Sub(start).Seconds())
	if secs < 0 {
		return nil
	}
	return &secs
}

// RecurrenceToRRule converts Google recurrence lines to a single RRULE string (without RRULE: prefix).
func RecurrenceToRRule(recurrence []string) *string {
	for _, line := range recurrence {
		upper := strings.ToUpper(strings.TrimSpace(line))
		if strings.HasPrefix(upper, "RRULE:") {
			rule := strings.TrimSpace(line[len("RRULE:"):])
			if rule == "" {
				continue
			}
			return &rule
		}
	}
	return nil
}

// RRuleToRecurrence wraps an RRULE string for Google Calendar API.
func RRuleToRecurrence(rrule *string) []string {
	if rrule == nil || strings.TrimSpace(*rrule) == "" {
		return nil
	}
	rule := strings.TrimSpace(*rrule)
	if strings.HasPrefix(strings.ToUpper(rule), "RRULE:") {
		return []string{rule}
	}
	return []string{"RRULE:" + rule}
}

// MapEventToTaskFields maps a Google event onto task fields for import upsert.
// messengerRelatedUserID is copied from the binding when set (needed for worker reminders).
// Recurring series with a past DTSTART advance start_date to the next occurrence so the task
// stays useful without autoreschedule (imported tasks are excluded from that job).
// One-shot events whose start is already past become status=done (Google does not "complete" them).
func MapEventToTaskFields(ev googlecalendar.Event, userID int64, groupID *int64, messengerRelatedUserID *int) (*models.Task, error) {
	start, err := EventStartTime(ev)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(ev.Summary)
	if title == "" {
		title = "(untitled event)"
	}
	rrule := RecurrenceToRRule(ev.Recurrence)
	now := time.Now().UTC()
	if rrule != nil && !start.After(now) {
		next, nextErr := nextStartFromRRule(now, *rrule, start)
		if nextErr == nil {
			start = next
		}
		// If the series has no future occurrence (COUNT/UNTIL exhausted), keep past start → done below.
	}

	task := &models.Task{
		Title:                  title,
		Description:            ev.Description,
		UserID:                 userID,
		GroupID:                groupID,
		MessengerRelatedUserID: messengerRelatedUserID,
		StartDate:              start,
		// FinishDate is completion time in GoReminder — duration lives on task_sync_links.
		RRule: rrule,
	}
	if start.After(now) {
		task.Status = string(models.TaskStatusScheduled)
	} else {
		// Past one-shot (or exhausted series): treat as completed, not overdue scheduled.
		task.Status = string(models.TaskStatusDone)
		finish := now
		if end := EventEndTime(ev); end != nil && !end.After(now) {
			finish = end.UTC()
		} else if !start.After(now) {
			finish = start
		}
		task.FinishDate = &finish
	}
	return task, nil
}

// ApplyDeletePolicy applies binding delete_policy to an imported task.
func ApplyDeletePolicy(policy models.CalendarDeletePolicy, task *models.Task) (softDelete bool, mute bool) {
	switch policy {
	case models.CalendarDeletePolicyMuteImported:
		return false, true
	case models.CalendarDeletePolicyKeep:
		return false, false
	default:
		return true, false
	}
}

// ShouldExportTask reports whether a task is eligible for calendar export.
// Confirmation children (parent_id set) are not exported separately.
func ShouldExportTask(task *models.Task) bool {
	if task == nil {
		return false
	}
	if task.ParentID != nil {
		return false
	}
	return true
}

// TaskIDFromExtendedProperties reads goreminder_task_id from private extended properties.
func TaskIDFromExtendedProperties(props map[string]string) (int64, bool) {
	if props == nil {
		return 0, false
	}
	raw, ok := props[googlecalendar.PrivateExtendedPropertyTaskID]
	if !ok || raw == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// BuildExportEvent builds a Google event payload from a task.
func BuildExportEvent(task *models.Task, durationMinutes int, existingDurationSeconds *int) *googlecalendar.Event {
	if durationMinutes <= 0 {
		durationMinutes = 30
	}
	start := task.StartDate.UTC()
	dur := time.Duration(durationMinutes) * time.Minute
	if existingDurationSeconds != nil && *existingDurationSeconds > 0 {
		dur = time.Duration(*existingDurationSeconds) * time.Second
	} else if task.FinishDate != nil && task.FinishDate.After(start) {
		dur = task.FinishDate.Sub(start)
	}
	end := start.Add(dur)
	ev := &googlecalendar.Event{
		Summary:     task.Title,
		Description: task.Description,
		Start:       googlecalendar.EventDateTime{DateTime: &start},
		End:         googlecalendar.EventDateTime{DateTime: &end},
		Recurrence:  RRuleToRecurrence(task.RRule),
		ExtendedProperties: map[string]string{
			googlecalendar.PrivateExtendedPropertyTaskID: strconv.FormatInt(task.ID, 10),
		},
	}
	return ev
}
