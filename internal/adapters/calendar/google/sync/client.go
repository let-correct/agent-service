package google

import (
	"context"
	"errors"
	"time"

	calendarsync "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

const (
	source        = "agent-service"
	schemaVersion = "1.0"
)

type Client struct {
	calendarService func(ctx context.Context, accessToken string) (CalendarService, error)
	now             func() time.Time
}

func NewClient() *Client {
	return &Client{
		calendarService: newGoogleCalendarService,
		now:             time.Now,
	}
}

func (c *Client) SyncEvents(ctx context.Context, email, accessToken, calendarID, syncToken string, lastSyncedAt time.Time) ([]calendarsync.Event, string, error) {
	svc, err := c.calendarService(ctx, accessToken)
	if err != nil {
		return nil, "", err
	}

	items, newSyncToken, err := svc.ListEvents(ctx, calendarID, syncToken, lastSyncedAt)
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 410 {
			return nil, "", calendarsync.ErrSyncTokenExpired
		}
		return nil, "", err
	}

	now := c.now().UTC()
	events := make([]calendarsync.Event, 0, len(items))
	for _, item := range items {
		events = append(events, mapEvent(item, email, calendarID, now))
	}

	return events, newSyncToken, nil
}

func mapEvent(item *calendar.Event, email, calendarID string, now time.Time) calendarsync.Event {
	detailType := calendarsync.DetailTypeCreated
	if item.Status == "cancelled" {
		detailType = calendarsync.DetailTypeCancelled
	}

	attendees := make([]string, 0, len(item.Attendees))
	for _, a := range item.Attendees {
		attendees = append(attendees, a.Email)
	}

	return calendarsync.Event{
		DetailType: detailType,
		Metadata: calendarsync.EventMetadata{
			Timestamp:     now,
			Source:        source,
			SchemaVersion: schemaVersion,
			CorrelationID: item.Id,
		},
		Payload: calendarsync.Appointment{
			Email:       email,
			EventID:     item.Id,
			CalendarID:  calendarID,
			Summary:     item.Summary,
			Description: item.Description,
			Start:       parseEventTime(item.Start),
			End:         parseEventTime(item.End),
			Attendees:   attendees,
			Status:      item.Status,
		},
	}
}

func parseEventTime(et *calendar.EventDateTime) time.Time {
	if et == nil {
		return time.Time{}
	}
	if et.DateTime != "" {
		t, _ := time.Parse(time.RFC3339, et.DateTime)
		return t
	}
	if et.Date != "" {
		t, _ := time.Parse("2006-01-02", et.Date)
		return t
	}
	return time.Time{}
}
