package service

import (
	"fmt"
	"strings"
	"time"

	rrulelib "github.com/teambition/rrule-go"
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
