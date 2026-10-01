package googlecalendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Calendar describes a Google calendar list entry.
type Calendar struct {
	ID      string
	Summary string
	Primary bool
}

// EventDateTime is a start/end instant (date-time or all-day date).
type EventDateTime struct {
	DateTime *time.Time
	Date     string // YYYY-MM-DD for all-day
	TimeZone string
}

// Event is a simplified Google Calendar event.
type Event struct {
	ID                 string
	Summary            string
	Description        string
	Start              EventDateTime
	End                EventDateTime
	Status             string
	ETag               string
	Updated            time.Time
	Recurrence         []string
	ExtendedProperties map[string]string // private properties
}

// ListEventsOpts controls Events.list parameters.
type ListEventsOpts struct {
	SyncToken string
	TimeMin   *time.Time
	TimeMax   *time.Time
	PageToken string
	MaxResults int64
}

// EventListResult is a page of events plus sync/page tokens.
type EventListResult struct {
	Events        []Event
	NextPageToken string
	NextSyncToken string
}

// CalendarClient abstracts Google Calendar API operations used by sync.
type CalendarClient interface {
	ListCalendars(ctx context.Context) ([]Calendar, error)
	ListEvents(ctx context.Context, calendarID string, opts ListEventsOpts) (*EventListResult, error)
	GetEvent(ctx context.Context, calendarID, eventID string) (*Event, error)
	CreateEvent(ctx context.Context, calendarID string, event *Event) (*Event, error)
	PatchEvent(ctx context.Context, calendarID, eventID string, event *Event) (*Event, error)
	DeleteEvent(ctx context.Context, calendarID, eventID string) error
}

// ClientFactory creates a CalendarClient authenticated with the given token.
type ClientFactory func(ctx context.Context, token *oauth2.Token) (CalendarClient, error)

// DefaultClientFactory returns a factory backed by the Google Calendar v3 API.
func DefaultClientFactory(oauthCfg *oauth2.Config) ClientFactory {
	return func(ctx context.Context, token *oauth2.Token) (CalendarClient, error) {
		ts := oauthCfg.TokenSource(ctx, token)
		svc, err := calendar.NewService(ctx, option.WithTokenSource(ts))
		if err != nil {
			return nil, fmt.Errorf("create calendar service: %w", err)
		}
		return &apiClient{svc: svc}, nil
	}
}

type apiClient struct {
	svc *calendar.Service
}

func (c *apiClient) ListCalendars(ctx context.Context) ([]Calendar, error) {
	var out []Calendar
	err := c.svc.CalendarList.List().Context(ctx).Pages(ctx, func(page *calendar.CalendarList) error {
		for _, item := range page.Items {
			out = append(out, Calendar{
				ID:      item.Id,
				Summary: item.Summary,
				Primary: item.Primary,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list calendars: %w", err)
	}
	return out, nil
}

func (c *apiClient) ListEvents(ctx context.Context, calendarID string, opts ListEventsOpts) (*EventListResult, error) {
	call := c.svc.Events.List(calendarID).Context(ctx).SingleEvents(false).ShowDeleted(true)
	if opts.SyncToken != "" {
		call = call.SyncToken(opts.SyncToken)
	} else {
		if opts.TimeMin != nil {
			call = call.TimeMin(opts.TimeMin.UTC().Format(time.RFC3339))
		}
		if opts.TimeMax != nil {
			call = call.TimeMax(opts.TimeMax.UTC().Format(time.RFC3339))
		}
	}
	if opts.PageToken != "" {
		call = call.PageToken(opts.PageToken)
	}
	if opts.MaxResults > 0 {
		call = call.MaxResults(opts.MaxResults)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	result := &EventListResult{
		NextPageToken: resp.NextPageToken,
		NextSyncToken: resp.NextSyncToken,
	}
	for _, item := range resp.Items {
		result.Events = append(result.Events, mapAPIEvent(item))
	}
	return result, nil
}

func (c *apiClient) GetEvent(ctx context.Context, calendarID, eventID string) (*Event, error) {
	item, err := c.svc.Events.Get(calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	ev := mapAPIEvent(item)
	return &ev, nil
}

func (c *apiClient) CreateEvent(ctx context.Context, calendarID string, event *Event) (*Event, error) {
	item, err := c.svc.Events.Insert(calendarID, toAPIEvent(event)).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}
	ev := mapAPIEvent(item)
	return &ev, nil
}

func (c *apiClient) PatchEvent(ctx context.Context, calendarID, eventID string, event *Event) (*Event, error) {
	item, err := c.svc.Events.Patch(calendarID, eventID, toAPIEvent(event)).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("patch event: %w", err)
	}
	ev := mapAPIEvent(item)
	return &ev, nil
}

func (c *apiClient) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	err := c.svc.Events.Delete(calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	return nil
}

func mapAPIEvent(item *calendar.Event) Event {
	ev := Event{
		ID:          item.Id,
		Summary:     item.Summary,
		Description: item.Description,
		Status:      item.Status,
		ETag:        item.Etag,
		Recurrence:  item.Recurrence,
	}
	if item.Updated != "" {
		if t, err := time.Parse(time.RFC3339, item.Updated); err == nil {
			ev.Updated = t
		}
	}
	ev.Start = mapEventDateTime(item.Start)
	ev.End = mapEventDateTime(item.End)
	if item.ExtendedProperties != nil && item.ExtendedProperties.Private != nil {
		ev.ExtendedProperties = item.ExtendedProperties.Private
	}
	return ev
}

func mapEventDateTime(dt *calendar.EventDateTime) EventDateTime {
	if dt == nil {
		return EventDateTime{}
	}
	out := EventDateTime{Date: dt.Date, TimeZone: dt.TimeZone}
	if dt.DateTime != "" {
		if t, err := time.Parse(time.RFC3339, dt.DateTime); err == nil {
			out.DateTime = &t
		}
	}
	return out
}

func toAPIEvent(event *Event) *calendar.Event {
	item := &calendar.Event{
		Summary:     event.Summary,
		Description: event.Description,
		Status:      event.Status,
		Recurrence:  event.Recurrence,
		Start:       toAPIEventDateTime(event.Start),
		End:         toAPIEventDateTime(event.End),
	}
	if len(event.ExtendedProperties) > 0 {
		item.ExtendedProperties = &calendar.EventExtendedProperties{
			Private: event.ExtendedProperties,
		}
	}
	return item
}

func toAPIEventDateTime(dt EventDateTime) *calendar.EventDateTime {
	out := &calendar.EventDateTime{TimeZone: dt.TimeZone}
	if dt.DateTime != nil {
		out.DateTime = dt.DateTime.UTC().Format(time.RFC3339)
	} else if dt.Date != "" {
		out.Date = dt.Date
	}
	return out
}

// IsSyncTokenInvalid reports whether err indicates a stale sync token (HTTP 410).
func IsSyncTokenInvalid(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "410") ||
		strings.Contains(msg, "Sync token is no longer valid") ||
		strings.Contains(msg, "fullSyncRequired")
}
