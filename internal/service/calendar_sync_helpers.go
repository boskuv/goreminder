package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/googlecalendar"
)

const (
	OutboxKindExportUpsert = "export_upsert"
	OutboxKindExportDelete = "export_delete"

	maxOutboxAttempts = 8
	// MaxBindingSyncAttempts caps automatic import retries for status=error bindings.
	// Force sync still runs regardless; success resets the counter.
	MaxBindingSyncAttempts = 8
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

// BindingSyncBackoff returns the delay before the next automatic import sync retry.
// Same schedule as OutboxBackoff (1m → 5m → 30m → 2h).
func BindingSyncBackoff(attempts int) time.Duration {
	return OutboxBackoff(attempts)
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

// CronExpressionToRRule maps a standard 5-field cron (min hour dom month dow) to an
// iCalendar RRULE body (without "RRULE:" prefix) for Google Calendar export.
// Event time-of-day comes from DTSTART; this encodes frequency / weekday / month-day.
// Returns nil when the expression cannot be represented safely as RRULE.
func CronExpressionToRRule(cronExpr string) *string {
	cronExpr = strings.TrimSpace(cronExpr)
	if cronExpr == "" {
		return nil
	}
	// Validate with the same parser the API uses.
	if _, err := cron.ParseStandard(cronExpr); err != nil {
		return nil
	}
	fields := strings.Fields(cronExpr)
	if len(fields) != 5 {
		return nil
	}
	minute, hour, dom, month, dow := fields[0], fields[1], fields[2], fields[3], fields[4]

	// Time-of-day is taken from DTSTART; reject odd step patterns on minute/hour that
	// would need MINUTELY/HOURLY series mismatched with a timed DTSTART.
	if strings.HasPrefix(minute, "*/") || strings.HasPrefix(hour, "*/") {
		return nil
	}
	if strings.ContainsAny(minute, "-,") || strings.ContainsAny(hour, "-,") {
		return nil
	}

	var rule string
	switch {
	case dom == "*" && month == "*" && dow == "*":
		rule = "FREQ=DAILY"
	case dom == "*" && month == "*" && dow != "*":
		byday, ok := cronDowToByDay(dow)
		if !ok {
			return nil
		}
		rule = "FREQ=WEEKLY;BYDAY=" + byday
	case dom != "*" && month == "*" && dow == "*":
		byMonthDay, ok := cronFieldToCSV(dom, 1, 31)
		if !ok {
			return nil
		}
		rule = "FREQ=MONTHLY;BYMONTHDAY=" + byMonthDay
	case dom != "*" && month != "*" && dow == "*":
		byMonthDay, ok := cronFieldToCSV(dom, 1, 31)
		if !ok {
			return nil
		}
		byMonth, ok := cronFieldToCSV(month, 1, 12)
		if !ok {
			return nil
		}
		rule = "FREQ=YEARLY;BYMONTH=" + byMonth + ";BYMONTHDAY=" + byMonthDay
	default:
		// Combined DOM+DOW (cron OR semantics) and other mixes are not mapped.
		return nil
	}
	return &rule
}

func cronFieldToCSV(field string, min, max int) (string, bool) {
	parts, ok := expandCronIntList(field, min, max)
	if !ok || len(parts) == 0 {
		return "", false
	}
	out := make([]string, len(parts))
	for i, n := range parts {
		out[i] = strconv.Itoa(n)
	}
	return strings.Join(out, ","), true
}

func expandCronIntList(field string, min, max int) ([]int, bool) {
	if field == "*" || strings.HasPrefix(field, "*/") {
		return nil, false
	}
	var result []int
	for _, piece := range strings.Split(field, ",") {
		piece = strings.TrimSpace(piece)
		if piece == "" {
			return nil, false
		}
		if strings.Contains(piece, "-") {
			bounds := strings.Split(piece, "-")
			if len(bounds) != 2 {
				return nil, false
			}
			lo, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err1 != nil || err2 != nil || lo > hi || lo < min || hi > max {
				return nil, false
			}
			for n := lo; n <= hi; n++ {
				result = append(result, n)
			}
			continue
		}
		n, err := strconv.Atoi(piece)
		if err != nil || n < min || n > max {
			return nil, false
		}
		result = append(result, n)
	}
	return result, true
}

// cronDowToByDay converts cron day-of-week (0-7, names, lists, ranges) to Google BYDAY.
func cronDowToByDay(field string) (string, bool) {
	if field == "*" || strings.HasPrefix(field, "*/") {
		return "", false
	}
	var nums []int
	for _, piece := range strings.Split(field, ",") {
		piece = strings.TrimSpace(piece)
		if piece == "" {
			return "", false
		}
		if strings.Contains(piece, "-") {
			bounds := strings.Split(piece, "-")
			if len(bounds) != 2 {
				return "", false
			}
			lo, ok1 := cronDowTokenToNumber(bounds[0])
			hi, ok2 := cronDowTokenToNumber(bounds[1])
			if !ok1 || !ok2 || lo > hi {
				return "", false
			}
			for n := lo; n <= hi; n++ {
				nums = append(nums, n)
			}
			continue
		}
		n, ok := cronDowTokenToNumber(piece)
		if !ok {
			return "", false
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return "", false
	}
	names := make([]string, 0, len(nums))
	seen := map[string]struct{}{}
	for _, n := range nums {
		if n == 7 {
			n = 0
		}
		if n < 0 || n > 6 {
			return "", false
		}
		name := cronDowByDay[n]
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return strings.Join(names, ","), true
}

var cronDowByDay = []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"}

func cronDowTokenToNumber(tok string) (int, bool) {
	tok = strings.ToLower(strings.TrimSpace(tok))
	switch tok {
	case "0", "7", "sun", "sunday":
		return 0, true
	case "1", "mon", "monday":
		return 1, true
	case "2", "tue", "tuesday":
		return 2, true
	case "3", "wed", "wednesday":
		return 3, true
	case "4", "thu", "thursday":
		return 4, true
	case "5", "fri", "friday":
		return 5, true
	case "6", "sat", "saturday":
		return 6, true
	}
	n, err := strconv.Atoi(tok)
	if err != nil || n < 0 || n > 7 {
		return 0, false
	}
	return n, true
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

// bindingAllowsTaskExport reports whether local task changes may be pushed to Google for this binding.
func bindingAllowsTaskExport(direction models.CalendarBindingDirection) bool {
	return direction == models.CalendarBindingDirectionExport ||
		direction == models.CalendarBindingDirectionBoth
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
// Recurrence prefers task.RRule; if unset, cron_expression is mapped to RRULE when possible.
// Start/end always use timeZone UTC (task times are stored in UTC); Google requires timeZone
// especially for recurring events (400 Missing time zone definition otherwise).
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

	rrule := task.RRule
	if !recurrenceFieldSet(rrule) && recurrenceFieldSet(task.CronExpression) {
		rrule = CronExpressionToRRule(*task.CronExpression)
	}

	const exportTimeZone = "UTC"
	ev := &googlecalendar.Event{
		Summary:     task.Title,
		Description: task.Description,
		Start:       googlecalendar.EventDateTime{DateTime: &start, TimeZone: exportTimeZone},
		End:         googlecalendar.EventDateTime{DateTime: &end, TimeZone: exportTimeZone},
		Recurrence:  RRuleToRecurrence(rrule),
		ExtendedProperties: map[string]string{
			googlecalendar.PrivateExtendedPropertyTaskID: strconv.FormatInt(task.ID, 10),
		},
	}
	return ev
}
