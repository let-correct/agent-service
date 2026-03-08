package google

import (
	"context"
	"errors"
	"time"

	calendarsync "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	source        = "agent-service"
	schemaVersion = "1.0"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) SyncEvents(
	ctx context.Context,
	email, accessToken, calendarID, syncToken string,
	lastSyncedAt time.Time,
) ([]calendarsync.Event, string, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	httpClient := oauth2.NewClient(ctx, tokenSource)

	svc, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, "", err
	}

	call := svc.Events.List(calendarID).Context(ctx).SingleEvents(true)
	if syncToken != "" {
		call = call.SyncToken(syncToken)
	} else if !lastSyncedAt.IsZero() {
		call = call.TimeMin(lastSyncedAt.Format(time.RFC3339))
	} else {
		call = call.TimeMin(time.Now().UTC().Format(time.RFC3339))
	}

	var events []calendarsync.Event
	var newSyncToken string

	err = call.Pages(ctx, func(page *calendar.Events) error {
		newSyncToken = page.NextSyncToken
		for _, item := range page.Items {
			events = append(events, mapEvent(item, email, calendarID))
		}
		return nil
	})
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 410 {
			return nil, "", calendarsync.ErrSyncTokenExpired
		}
		return nil, "", err
	}

	return events, newSyncToken, nil
}

func mapEvent(item *calendar.Event, email, calendarID string) calendarsync.Event {
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
			Timestamp:     time.Now().UTC(),
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
