package google

import (
	"context"
	"errors"
	"strings"
	"time"

	calendarsync "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

const (
	source        = "agent-service"
	schemaVersion = "1.0"

	// bookedByPrefix is the description prefix used to identify viewing appointments
	// and to locate the attendee block. Only events whose description starts with
	// this prefix are synced. To support additional appointment types in future,
	// extend isViewingAppointment accordingly.
	bookedByPrefix = "<b>Booked by</b>\n"
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
		// only currently emit viewing appointment events, but can be extended in the future
		if !isViewingAppointment(item) {
			continue
		}
		events = append(events, mapEvent(item, email, now))
	}

	return events, newSyncToken, nil
}

func isViewingAppointment(item *calendar.Event) bool {
	return strings.HasPrefix(item.Description, bookedByPrefix)
}

func mapEvent(item *calendar.Event, email string, now time.Time) calendarsync.Event {
	detailType := calendarsync.DetailTypeCreated
	if item.Status == "cancelled" {
		detailType = calendarsync.DetailTypeCancelled
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
			AgentEmail: email,
			Location:   item.Location,
			Start:      parseEventTime(item.Start),
			End:        parseEventTime(item.End),
			Attendee:   parseAttendee(item.Description),
		},
	}
}

// parseAttendee extracts the attendee name, email, and phone from the
// description block that follows the "Booked by" prefix. Lines are expected
// in the order: name, email, phone. Missing lines result in empty fields;
// the event is still emitted.
func parseAttendee(description string) calendarsync.Attendee {
	body := strings.TrimPrefix(description, bookedByPrefix)
	lines := strings.Split(body, "\n")

	var a calendarsync.Attendee
	if len(lines) > 0 {
		a.Name = lines[0]
	}
	if len(lines) > 1 {
		a.Email = lines[1]
	}
	if len(lines) > 2 {
		a.Phone = lines[2]
	}
	return a
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
