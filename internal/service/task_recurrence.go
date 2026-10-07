package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"
	rrulelib "github.com/teambition/rrule-go"

	"github.com/boskuv/goreminder/internal/models"
)

// errRecurrenceNotShiftable means the cron or RRULE is not a single interval that can be
// re-anchored to a completion date (lists, steps, COUNT, UNTIL, and similar).
var errRecurrenceNotShiftable = errors.New("recurrence rule cannot be shifted from the completion date")

func recurrenceFieldSet(p *string) bool {
	return p != nil && strings.TrimSpace(*p) != ""
}

// recurrenceFieldChanged reports whether a recurrence field was explicitly changed in a partial update.
// Nil means "field not provided", so it's not a change.
// Empty string and nil are treated as equivalent "not set" values.
func recurrenceFieldChanged(updated, current *string) bool {
	if updated == nil {
		return false
	}

	updatedValue := strings.TrimSpace(*updated)
	currentValue := ""
	if current != nil {
		currentValue = strings.TrimSpace(*current)
	}

	return updatedValue != currentValue
}

func validateCronExpressionAndRRuleExclusive(cronExpr, rrule *string) error {
	if recurrenceFieldSet(cronExpr) && recurrenceFieldSet(rrule) {
		return fmt.Errorf("cron_expression and rrule cannot both be set")
	}
	return nil
}

func validateRRuleString(rruleStr string, seriesStart time.Time) error {
	opt, err := rrulelib.StrToROption(strings.TrimSpace(rruleStr))
	if err != nil {
		return err
	}
	if opt.Dtstart.IsZero() {
		opt.Dtstart = seriesStart.UTC()
	}
	_, err = rrulelib.NewRRule(*opt)
	return err
}

func nextStartFromRRule(now time.Time, rruleStr string, seriesStart time.Time) (time.Time, error) {
	opt, err := rrulelib.StrToROption(strings.TrimSpace(rruleStr))
	if err != nil {
		return time.Time{}, err
	}
	if opt.Dtstart.IsZero() {
		opt.Dtstart = seriesStart.UTC()
	}
	rule, err := rrulelib.NewRRule(*opt)
	if err != nil {
		return time.Time{}, err
	}
	next := rule.After(now.UTC(), false)
	if next.IsZero() {
		return time.Time{}, fmt.Errorf("no occurrence after %s", now.UTC().Format(time.RFC3339))
	}
	return next.UTC(), nil
}

// isRecurrenceParentWithConfirmation identifies tasks that keep the recurrence rule on the parent row
// while child tasks carry the next executable occurrence (same model as cron + requires_confirmation).
func isRecurrenceParentWithConfirmation(t *models.Task) bool {
	if t == nil || !t.RequiresConfirmation {
		return false
	}
	return recurrenceFieldSet(t.CronExpression) || recurrenceFieldSet(t.RRule)
}

// hasRecurrenceRule is true when the task row has either cron_expression or rrule set (non-empty).
func hasRecurrenceRule(t *models.Task) bool {
	if t == nil {
		return false
	}
	return recurrenceFieldSet(t.CronExpression) || recurrenceFieldSet(t.RRule)
}

// nextExecutableStartDate returns the start_date to use when publishing worker.schedule_task after unmute.
// If start_date is already in the future (or now), it is returned unchanged.
// For executable recurring rows (cron/rrule on the task, not confirmation parents), returns the next
// occurrence from the rule when start_date is in the past (same calculation as RescheduleCronTasks).
// For child tasks with a parent recurrence rule, returns the parent's next occurrence after now.
// For one-time tasks with start_date in the past, returns zero time.
func nextExecutableStartDate(task *models.Task, parent *models.Task, now time.Time) (time.Time, error) {
	if task == nil {
		return time.Time{}, fmt.Errorf("task is nil")
	}
	now = now.UTC()
	if !task.StartDate.Before(now) {
		return task.StartDate.UTC(), nil
	}
	if isRecurrenceParentWithConfirmation(task) {
		return time.Time{}, nil
	}
	if recurrenceFieldSet(task.CronExpression) {
		return cronexpr.MustParse(*task.CronExpression).Next(now), nil
	}
	if recurrenceFieldSet(task.RRule) {
		return nextStartFromRRule(now, *task.RRule, task.StartDate)
	}
	if task.ParentID != nil && parent != nil && hasRecurrenceRule(parent) {
		return nextRecurrenceAfter(parent, now)
	}
	return time.Time{}, nil
}

// nextRecurrenceAfter returns the first occurrence strictly after `from` using whichever recurrence
// field is set on the parent (cron and rrule are mutually exclusive).
func nextRecurrenceAfter(parent *models.Task, from time.Time) (time.Time, error) {
	if parent == nil {
		return time.Time{}, fmt.Errorf("parent task is nil")
	}
	fromUTC := from.UTC()
	if recurrenceFieldSet(parent.CronExpression) {
		nextTime := cronexpr.MustParse(*parent.CronExpression).Next(fromUTC)
		if nextTime.Before(fromUTC) {
			nextTime = cronexpr.MustParse(*parent.CronExpression).Next(nextTime)
		}
		return nextTime.UTC(), nil
	}
	if recurrenceFieldSet(parent.RRule) {
		return nextStartFromRRule(fromUTC, *parent.RRule, parent.StartDate)
	}
	return time.Time{}, fmt.Errorf("parent task has no recurrence rule")
}

// completionShift is a confirmation series re-anchored to a completion instant.
// Anchor is the new parent start_date (completion calendar day, clock from the completed child).
// NextStart is the first occurrence strictly after completedAt on the shifted rule.
type completionShift struct {
	Anchor    time.Time
	NextStart time.Time
	Cron      *string
	RRule     *string
}

// shiftRecurrenceFromCompletion rewrites a confirmation parent's cron or RRULE so later
// occurrences follow the completion date. The stored rule string is left unchanged when it
// has no BYDAY / BYMONTHDAY / BYMONTH; the new day then comes from Anchor as DTSTART.
// Clock time on Anchor comes from clock (the completed child's start_date). Cron next-fire
// times still use the minute and hour in the expression.
func shiftRecurrenceFromCompletion(parent *models.Task, clock, completedAt time.Time) (completionShift, error) {
	if parent == nil || !hasRecurrenceRule(parent) {
		return completionShift{}, errRecurrenceNotShiftable
	}
	if recurrenceFieldSet(parent.CronExpression) && recurrenceFieldSet(parent.RRule) {
		return completionShift{}, errRecurrenceNotShiftable
	}

	anchor := completionAnchor(completedAt, clock)
	shifted := *parent
	shifted.StartDate = anchor

	if recurrenceFieldSet(parent.CronExpression) {
		cronExpr, err := shiftCronFromCompletion(*parent.CronExpression, anchor)
		if err != nil {
			return completionShift{}, err
		}
		if _, err := cronexpr.Parse(cronExpr); err != nil {
			return completionShift{}, errRecurrenceNotShiftable
		}
		shifted.CronExpression = &cronExpr
		next, err := nextRecurrenceAfter(&shifted, completedAt)
		if err != nil {
			return completionShift{}, errRecurrenceNotShiftable
		}
		return completionShift{Anchor: anchor, NextStart: next, Cron: &cronExpr}, nil
	}

	rruleExpr, err := shiftRRuleFromCompletion(*parent.RRule, anchor)
	if err != nil {
		return completionShift{}, err
	}
	if err := validateRRuleString(rruleExpr, anchor); err != nil {
		return completionShift{}, errRecurrenceNotShiftable
	}
	shifted.RRule = &rruleExpr
	next, err := nextRecurrenceAfter(&shifted, completedAt)
	if err != nil {
		return completionShift{}, errRecurrenceNotShiftable
	}
	return completionShift{Anchor: anchor, NextStart: next, RRule: &rruleExpr}, nil
}

func completionAnchor(completedAt, clock time.Time) time.Time {
	completedAt = completedAt.UTC()
	clock = clock.UTC()
	return time.Date(completedAt.Year(), completedAt.Month(), completedAt.Day(), clock.Hour(), clock.Minute(), clock.Second(), 0, time.UTC)
}

func shiftCronFromCompletion(expr string, anchor time.Time) (string, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return "", errRecurrenceNotShiftable
	}
	minute, hour, dom, month, dow := fields[0], fields[1], fields[2], fields[3], fields[4]
	if !singleCronNumber(minute, 0, 59) || !singleCronNumber(hour, 0, 23) {
		return "", errRecurrenceNotShiftable
	}

	switch {
	case dom == "*" && month == "*" && dow == "*":
		return strings.Join(fields, " "), nil
	case dom == "*" && month == "*" && singleCronNumber(dow, 0, 7):
		return fmt.Sprintf("%s %s * * %d", minute, hour, int(anchor.Weekday())), nil
	case month == "*" && dow == "*" && singleCronNumber(dom, 1, 31):
		return fmt.Sprintf("%s %s %d * *", minute, hour, anchor.Day()), nil
	case dow == "*" && singleCronNumber(dom, 1, 31) && singleCronNumber(month, 1, 12):
		return fmt.Sprintf("%s %s %d %d *", minute, hour, anchor.Day(), int(anchor.Month())), nil
	default:
		return "", errRecurrenceNotShiftable
	}
}

func singleCronNumber(field string, low, high int) bool {
	if strings.ContainsAny(field, ",-*/") {
		return false
	}
	n, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return n >= low && n <= high
}

func shiftRRuleFromCompletion(rruleStr string, anchor time.Time) (string, error) {
	raw := strings.TrimSpace(rruleStr)
	if strings.Contains(raw, "\n") || strings.Contains(strings.ToUpper(raw), "DTSTART") {
		return "", errRecurrenceNotShiftable
	}
	opt, err := rrulelib.StrToROption(raw)
	if err != nil {
		return "", errRecurrenceNotShiftable
	}
	switch opt.Freq {
	case rrulelib.DAILY, rrulelib.WEEKLY, rrulelib.MONTHLY, rrulelib.YEARLY:
	default:
		return "", errRecurrenceNotShiftable
	}
	if opt.Interval < 0 || opt.Count != 0 || !opt.Until.IsZero() {
		return "", errRecurrenceNotShiftable
	}
	if len(opt.Bysetpos) > 0 || len(opt.Byyearday) > 0 || len(opt.Byweekno) > 0 ||
		len(opt.Byhour) > 0 || len(opt.Byminute) > 0 || len(opt.Bysecond) > 0 || len(opt.Byeaster) > 0 {
		return "", errRecurrenceNotShiftable
	}
	if len(opt.Byweekday) > 1 || len(opt.Bymonthday) > 1 || len(opt.Bymonth) > 1 {
		return "", errRecurrenceNotShiftable
	}
	if len(opt.Byweekday) == 1 && opt.Byweekday[0].N() != 0 {
		return "", errRecurrenceNotShiftable
	}

	repl := map[string]string{}
	if len(opt.Byweekday) == 1 {
		repl["BYDAY"] = rruleWeekdayToken(anchor.Weekday())
	}
	if len(opt.Bymonthday) == 1 {
		repl["BYMONTHDAY"] = strconv.Itoa(anchor.Day())
	}
	if len(opt.Bymonth) == 1 {
		repl["BYMONTH"] = strconv.Itoa(int(anchor.Month()))
	}
	if len(repl) == 0 {
		return strings.TrimPrefix(raw, "RRULE:"), nil
	}
	return replaceRRuleParts(raw, repl)
}

func rruleWeekdayToken(wd time.Weekday) string {
	tokens := []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"}
	return tokens[wd]
}

func replaceRRuleParts(raw string, repl map[string]string) (string, error) {
	body := strings.TrimPrefix(strings.TrimSpace(raw), "RRULE:")
	parts := strings.Split(body, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		key, _, ok := strings.Cut(part, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return "", errRecurrenceNotShiftable
		}
		upper := strings.ToUpper(strings.TrimSpace(key))
		if next, exists := repl[upper]; exists {
			out = append(out, upper+"="+next)
			delete(repl, upper)
			continue
		}
		out = append(out, part)
	}
	if len(repl) != 0 {
		return "", errRecurrenceNotShiftable
	}
	return strings.Join(out, ";"), nil
}
