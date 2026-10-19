// Package ics parses iCalendar (.ics) files into calendar events for GoReminder import.
// v1: VEVENT only; VTODO/VJOURNAL/VALARM are ignored.
package ics

import (
	"fmt"
	"io"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"
)

// Event is a simplified VEVENT suitable for mapping into a GoReminder task
// (same shape as a Google Calendar event for MapEventToTaskFields).
type Event struct {
	UID         string
	Summary     string
	Description string
	Start       time.Time
	End         *time.Time
	AllDay      bool
	// Recurrence holds RRULE lines as Google Calendar does ("RRULE:FREQ=...").
	Recurrence []string
}

// ParseResult is the outcome of parsing one .ics document.
type ParseResult struct {
	Events      []Event
	SkippedTODO int
	Errors      []ParseError
}

// ParseError describes a single VEVENT that could not be mapped.
type ParseError struct {
	Index   int    // 0-based index among VEVENT components
	UID     string
	Message string
}

// Parse reads an iCalendar stream and returns VEVENT entries.
// VTODO components are counted in SkippedTODO and otherwise ignored.
// A broken VEVENT becomes a ParseError; other events are still returned.
func Parse(r io.Reader) (*ParseResult, error) {
	cal, err := ical.ParseCalendar(r)
	if err != nil {
		return nil, fmt.Errorf("parse calendar: %w", err)
	}

	out := &ParseResult{
		SkippedTODO: len(cal.Todos()),
	}

	for i, ev := range cal.Events() {
		parsed, perr := mapVEvent(i, ev)
		if perr != nil {
			out.Errors = append(out.Errors, *perr)
			continue
		}
		out.Events = append(out.Events, *parsed)
	}
	return out, nil
}

func mapVEvent(index int, ev *ical.VEvent) (*Event, *ParseError) {
	uid := ""
	if p := ev.GetProperty(ical.ComponentPropertyUniqueId); p != nil {
		uid = strings.TrimSpace(p.Value)
	}

	allDay := isAllDayDTStart(ev)

	var start time.Time
	var err error
	if allDay {
		start, err = allDayDateUTC(ev, ical.ComponentPropertyDtStart)
		if err != nil {
			// Fall back to GetStartAt and pin calendar day in that location before UTC.
			start, err = ev.GetStartAt()
			if err != nil {
				return nil, &ParseError{Index: index, UID: uid, Message: fmt.Sprintf("DTSTART: %v", err)}
			}
			start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
		}
	} else {
		start, err = ev.GetStartAt()
		if err != nil {
			return nil, &ParseError{Index: index, UID: uid, Message: fmt.Sprintf("DTSTART: %v", err)}
		}
		start = start.UTC()
	}

	var endPtr *time.Time
	if allDay {
		if end, endErr := allDayDateUTC(ev, ical.ComponentPropertyDtEnd); endErr == nil {
			endPtr = &end
		}
	} else if end, endErr := ev.GetEndAt(); endErr == nil {
		end = end.UTC()
		endPtr = &end
	}

	summary := ""
	if p := ev.GetProperty(ical.ComponentPropertySummary); p != nil {
		summary = p.Value
	}
	description := ""
	if p := ev.GetProperty(ical.ComponentPropertyDescription); p != nil {
		description = p.Value
	}

	var recurrence []string
	for _, p := range ev.GetProperties(ical.ComponentPropertyRrule) {
		val := strings.TrimSpace(p.Value)
		if val == "" {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(val), "RRULE:") {
			recurrence = append(recurrence, val)
		} else {
			recurrence = append(recurrence, "RRULE:"+val)
		}
		break // first RRULE only (Google parity)
	}

	return &Event{
		UID:         uid,
		Summary:     summary,
		Description: description,
		Start:       start,
		End:         endPtr,
		AllDay:      allDay,
		Recurrence:  recurrence,
	}, nil
}

func isAllDayDTStart(ev *ical.VEvent) bool {
	p := ev.GetProperty(ical.ComponentPropertyDtStart)
	if p == nil {
		return false
	}
	for k, vals := range p.ICalParameters {
		if !strings.EqualFold(k, string(ical.ParameterValue)) {
			continue
		}
		for _, v := range vals {
			if strings.EqualFold(v, string(ical.ValueDataTypeDate)) {
				return true
			}
		}
	}
	// DATE-only form YYYYMMDD (no time component).
	val := strings.TrimSpace(p.Value)
	return len(val) == 8 && !strings.ContainsAny(val, "T:")
}

// allDayDateUTC reads a DATE property as UTC midnight for that calendar day
// (avoids local-TZ skew from the ICS library's GetStartAt).
func allDayDateUTC(ev *ical.VEvent, prop ical.ComponentProperty) (time.Time, error) {
	p := ev.GetProperty(prop)
	if p == nil {
		return time.Time{}, fmt.Errorf("missing %s", prop)
	}
	val := strings.TrimSpace(p.Value)
	if len(val) < 8 {
		return time.Time{}, fmt.Errorf("invalid DATE value %q", val)
	}
	return time.ParseInLocation("20060102", val[:8], time.UTC)
}
