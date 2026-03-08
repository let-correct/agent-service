package google

import (
	"context"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// CalendarService abstracts Google Calendar API interactions.
type CalendarService interface {
	ListEvents(ctx context.Context, calendarID, syncToken string, timeMin time.Time) ([]*calendar.Event, string, error)
}

type googleCalendarService struct {
	svc *calendar.Service
}

func newGoogleCalendarService(ctx context.Context, accessToken string) (CalendarService, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	httpClient := oauth2.NewClient(ctx, tokenSource)
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}
	return &googleCalendarService{svc: svc}, nil
}

func (g *googleCalendarService) ListEvents(ctx context.Context, calendarID, syncToken string, timeMin time.Time) ([]*calendar.Event, string, error) {
	call := g.svc.Events.List(calendarID).Context(ctx).SingleEvents(true)
	if syncToken != "" {
		call = call.SyncToken(syncToken)
	} else if !timeMin.IsZero() {
		call = call.TimeMin(timeMin.Format(time.RFC3339))
	} else {
		call = call.TimeMin(time.Now().Add(-eventBuffer).UTC().Format(time.RFC3339))
	}

	var items []*calendar.Event
	var newSyncToken string

	err := call.Pages(ctx, func(page *calendar.Events) error {
		newSyncToken = page.NextSyncToken
		items = append(items, page.Items...)
		return nil
	})
	return items, newSyncToken, err
}
