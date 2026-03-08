package google

import (
	"context"
	"errors"
	"testing"
	"time"

	calendarsync "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

// mockCalendarService records calls to ListEvents for assertion in tests.
type mockCalendarService struct {
	calledWithSyncToken string
	calledWithTimeMin   time.Time
	items               []*calendar.Event
	newSyncToken        string
	err                 error
}

func (m *mockCalendarService) ListEvents(_ context.Context, _, syncToken string, timeMin time.Time) ([]*calendar.Event, string, error) {
	m.calledWithSyncToken = syncToken
	m.calledWithTimeMin = timeMin
	return m.items, m.newSyncToken, m.err
}

func newTestClient(svc *mockCalendarService, now func() time.Time) *Client {
	return &Client{
		calendarService: func(_ context.Context, _ string) (CalendarService, error) {
			return svc, nil
		},
		now: now,
	}
}

func TestSyncEvents(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	fixedNow := func() time.Time { return fixedTime }

	tests := []struct {
		name          string
		svc           *mockCalendarService
		factoryErr    error
		syncToken     string
		lastSyncedAt  time.Time
		wantErr       error
		wantTimeMin   time.Time
		wantSyncToken string
		wantEvents    []calendarsync.Event
	}{
		{
			name:       "service factory error is returned",
			factoryErr: errors.New("factory error"),
			wantErr:    errors.New("factory error"),
		},
		{
			name:    "ListEvents 410 returns ErrSyncTokenExpired",
			svc:     &mockCalendarService{err: &googleapi.Error{Code: 410}},
			wantErr: calendarsync.ErrSyncTokenExpired,
		},
		{
			name:    "ListEvents other error is propagated",
			svc:     &mockCalendarService{err: errors.New("api error")},
			wantErr: errors.New("api error"),
		},
		{
			name:          "with syncToken, forwards it and leaves timeMin zero",
			svc:           &mockCalendarService{newSyncToken: "new-token"},
			syncToken:     "existing-token",
			wantTimeMin:   time.Time{},
			wantSyncToken: "new-token",
		},
		{
			name:         "with lastSyncedAt and no syncToken, uses lastSyncedAt as timeMin",
			svc:          &mockCalendarService{},
			lastSyncedAt: fixedTime.Add(-48 * time.Hour),
			wantTimeMin:  fixedTime.Add(-48 * time.Hour),
		},
		{
			name:        "with neither syncToken nor lastSyncedAt, uses now minus eventBuffer",
			svc:         &mockCalendarService{},
			wantTimeMin: fixedTime.Add(-eventBuffer).UTC(),
		},
		{
			name:        "confirmed event is mapped with DetailTypeCreated",
			wantTimeMin: fixedTime.Add(-eventBuffer).UTC(),
			svc: &mockCalendarService{
				items: []*calendar.Event{
					{
						Id:      "evt1",
						Summary: "Standup",
						Status:  "confirmed",
						Attendees: []*calendar.EventAttendee{
							{Email: "a@example.com"},
							{Email: "b@example.com"},
						},
					},
				},
				newSyncToken: "tok",
			},
			wantSyncToken: "tok",
			wantEvents: []calendarsync.Event{
				{
					DetailType: calendarsync.DetailTypeCreated,
					Metadata: calendarsync.EventMetadata{
						Timestamp:     fixedTime.UTC(),
						Source:        source,
						SchemaVersion: schemaVersion,
						CorrelationID: "evt1",
					},
					Payload: calendarsync.Appointment{
						Email:      "user@example.com",
						EventID:    "evt1",
						CalendarID: "cal123",
						Summary:    "Standup",
						Status:     "confirmed",
						Attendees:  []string{"a@example.com", "b@example.com"},
					},
				},
			},
		},
		{
			name:        "cancelled event is mapped with DetailTypeCancelled",
			wantTimeMin: fixedTime.Add(-eventBuffer).UTC(),
			svc: &mockCalendarService{
				items: []*calendar.Event{
					{Id: "evt2", Status: "cancelled"},
				},
			},
			wantEvents: []calendarsync.Event{
				{
					DetailType: calendarsync.DetailTypeCancelled,
					Metadata: calendarsync.EventMetadata{
						Timestamp:     fixedTime.UTC(),
						Source:        source,
						SchemaVersion: schemaVersion,
						CorrelationID: "evt2",
					},
					Payload: calendarsync.Appointment{
						Email:      "user@example.com",
						EventID:    "evt2",
						CalendarID: "cal123",
						Status:     "cancelled",
						Attendees:  []string{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c *Client
			if tt.factoryErr != nil {
				c = &Client{
					calendarService: func(_ context.Context, _ string) (CalendarService, error) {
						return nil, tt.factoryErr
					},
					now: fixedNow,
				}
			} else {
				c = newTestClient(tt.svc, fixedNow)
			}

			events, newToken, err := c.SyncEvents(context.Background(), "user@example.com", "access-token", "cal123", tt.syncToken, tt.lastSyncedAt)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.svc != nil && !tt.svc.calledWithTimeMin.Equal(tt.wantTimeMin) {
				t.Errorf("timeMin: want %v, got %v", tt.wantTimeMin, tt.svc.calledWithTimeMin)
			}
			if tt.svc != nil && tt.svc.calledWithSyncToken != tt.syncToken {
				t.Errorf("syncToken forwarded: want %q, got %q", tt.syncToken, tt.svc.calledWithSyncToken)
			}
			if newToken != tt.wantSyncToken {
				t.Errorf("newSyncToken: want %q, got %q", tt.wantSyncToken, newToken)
			}

			if len(events) != len(tt.wantEvents) {
				t.Fatalf("event count: want %d, got %d", len(tt.wantEvents), len(events))
			}
			for i, want := range tt.wantEvents {
				got := events[i]
				if got.DetailType != want.DetailType {
					t.Errorf("events[%d].DetailType: want %q, got %q", i, want.DetailType, got.DetailType)
				}
				if !got.Metadata.Timestamp.Equal(want.Metadata.Timestamp) {
					t.Errorf("events[%d].Metadata.Timestamp: want %v, got %v", i, want.Metadata.Timestamp, got.Metadata.Timestamp)
				}
				if got.Metadata.Source != want.Metadata.Source {
					t.Errorf("events[%d].Metadata.Source: want %q, got %q", i, want.Metadata.Source, got.Metadata.Source)
				}
				if got.Metadata.CorrelationID != want.Metadata.CorrelationID {
					t.Errorf("events[%d].Metadata.CorrelationID: want %q, got %q", i, want.Metadata.CorrelationID, got.Metadata.CorrelationID)
				}
				p, wp := got.Payload, want.Payload
				if p.Email != wp.Email {
					t.Errorf("events[%d].Payload.Email: want %q, got %q", i, wp.Email, p.Email)
				}
				if p.EventID != wp.EventID {
					t.Errorf("events[%d].Payload.EventID: want %q, got %q", i, wp.EventID, p.EventID)
				}
				if p.CalendarID != wp.CalendarID {
					t.Errorf("events[%d].Payload.CalendarID: want %q, got %q", i, wp.CalendarID, p.CalendarID)
				}
				if p.Status != wp.Status {
					t.Errorf("events[%d].Payload.Status: want %q, got %q", i, wp.Status, p.Status)
				}
				if len(p.Attendees) != len(wp.Attendees) {
					t.Fatalf("events[%d].Payload.Attendees count: want %d, got %d", i, len(wp.Attendees), len(p.Attendees))
				}
				for j, email := range wp.Attendees {
					if p.Attendees[j] != email {
						t.Errorf("events[%d].Payload.Attendees[%d]: want %q, got %q", i, j, email, p.Attendees[j])
					}
				}
			}
		})
	}
}
