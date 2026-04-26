package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"
	rrulelib "github.com/teambition/rrule-go"

	"github.com/boskuv/goreminder/internal/models"
)

func recurrenceFieldSet(p *string) bool {
	return p != nil && strings.TrimSpace(*p) != ""
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
