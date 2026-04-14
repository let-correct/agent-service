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

const validDescription = "<b>Booked by</b>\nJohn Smith\njohn@example.com\n0821234567"

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
			name: "event without booked-by description prefix is filtered out",
			svc: &mockCalendarService{
				items: []*calendar.Event{
					{Id: "evt1", Summary: "Team Meeting", Status: "confirmed", Description: "Some other description"},
				},
				newSyncToken: "tok",
			},
			wantSyncToken: "tok",
			wantEvents:    []calendarsync.Event{},
		},
		{
			name: "event with empty description is filtered out",
			svc: &mockCalendarService{
				items: []*calendar.Event{
					{Id: "evt1", Summary: "Viewing: 10 Main St", Status: "confirmed", Description: ""},
				},
				newSyncToken: "tok",
			},
			wantSyncToken: "tok",
			wantEvents:    []calendarsync.Event{},
		},
		{
			name: "confirmed viewing appointment is mapped with DetailTypeCreated",
			svc: &mockCalendarService{
				items: []*calendar.Event{
					{
						Id:          "evt1",
						Summary:     "Viewing: 10 Main St",
						Status:      "confirmed",
						Description: validDescription,
						Location:    "10 Main St, Dublin",
						Start:       &calendar.EventDateTime{DateTime: "2024-01-20T10:00:00Z"},
						End:         &calendar.EventDateTime{DateTime: "2024-01-20T10:30:00Z"},
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
						AgentEmail: "user@example.com",
	

						Location:   "10 Main St, Dublin",
						Start:      time.Date(2024, 1, 20, 10, 0, 0, 0, time.UTC),
						End:        time.Date(2024, 1, 20, 10, 30, 0, 0, time.UTC),
						Attendee: calendarsync.Attendee{
							Name:  "John Smith",
							Email: "john@example.com",
							Phone: "0821234567",
						},
					},
				},
			},
		},
		{
			name: "cancelled viewing appointment is mapped with DetailTypeCancelled",
			svc: &mockCalendarService{
				items: []*calendar.Event{
					{
						Id:          "evt2",
						Summary:     "Viewing: 10 Main St",
						Status:      "cancelled",
						Description: validDescription,
					},
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
						AgentEmail: "user@example.com",

						Attendee: calendarsync.Attendee{
							Name:  "John Smith",
							Email: "john@example.com",
							Phone: "0821234567",
						},
					},
				},
			},
		},
		{
			name: "description with only name still emits event with partial attendee",
			svc: &mockCalendarService{
				items: []*calendar.Event{
					{
						Id:          "evt3",
						Summary:     "Viewing: 5 Oak Ave",
						Status:      "confirmed",
						Description: "<b>Booked by</b>\nJane Doe",
					},
				},
			},
			wantEvents: []calendarsync.Event{
				{
					DetailType: calendarsync.DetailTypeCreated,
					Metadata: calendarsync.EventMetadata{
						Timestamp:     fixedTime.UTC(),
						Source:        source,
						SchemaVersion: schemaVersion,
						CorrelationID: "evt3",
					},
					Payload: calendarsync.Appointment{
						AgentEmail: "user@example.com",

						Attendee: calendarsync.Attendee{
							Name: "Jane Doe",
						},
					},
				},
			},
		},
		{
			name: "description with name and email but no phone still emits event",
			svc: &mockCalendarService{
				items: []*calendar.Event{
					{
						Id:          "evt4",
						Summary:     "Viewing: 5 Oak Ave",
						Status:      "confirmed",
						Description: "<b>Booked by</b>\nJane Doe\njane@example.com",
					},
				},
			},
			wantEvents: []calendarsync.Event{
				{
					DetailType: calendarsync.DetailTypeCreated,
					Metadata: calendarsync.EventMetadata{
						Timestamp:     fixedTime.UTC(),
						Source:        source,
						SchemaVersion: schemaVersion,
						CorrelationID: "evt4",
					},
					Payload: calendarsync.Appointment{
						AgentEmail: "user@example.com",

						Attendee: calendarsync.Attendee{
							Name:  "Jane Doe",
							Email: "jane@example.com",
						},
					},
				},
			},
		},
		{
			name: "mixed events: only viewing appointments are emitted",
			svc: &mockCalendarService{
				items: []*calendar.Event{
					{Id: "evt1", Summary: "Viewing: 10 Main St", Status: "confirmed", Description: validDescription},
					{Id: "evt2", Summary: "Team Meeting", Status: "confirmed", Description: "Agenda: ..."},
					{Id: "evt3", Summary: "Viewing: 5 Oak Ave", Status: "confirmed", Description: validDescription},
				},
				newSyncToken: "tok",
			},
			wantSyncToken: "tok",
			wantEvents: []calendarsync.Event{
				{
					DetailType: calendarsync.DetailTypeCreated,
					Metadata:   calendarsync.EventMetadata{Timestamp: fixedTime.UTC(), Source: source, SchemaVersion: schemaVersion, CorrelationID: "evt1"},
					Payload:    calendarsync.Appointment{AgentEmail: "user@example.com", Attendee: calendarsync.Attendee{Name: "John Smith", Email: "john@example.com", Phone: "0821234567"}},
				},
				{
					DetailType: calendarsync.DetailTypeCreated,
					Metadata:   calendarsync.EventMetadata{Timestamp: fixedTime.UTC(), Source: source, SchemaVersion: schemaVersion, CorrelationID: "evt3"},
					Payload:    calendarsync.Appointment{AgentEmail: "user@example.com", Attendee: calendarsync.Attendee{Name: "John Smith", Email: "john@example.com", Phone: "0821234567"}},
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
				if p.AgentEmail != wp.AgentEmail {
					t.Errorf("events[%d].Payload.AgentEmail: want %q, got %q", i, wp.AgentEmail, p.AgentEmail)
				}
				if p.Location != wp.Location {
					t.Errorf("events[%d].Payload.Location: want %q, got %q", i, wp.Location, p.Location)
				}
				if !p.Start.Equal(wp.Start) {
					t.Errorf("events[%d].Payload.Start: want %v, got %v", i, wp.Start, p.Start)
				}
				if !p.End.Equal(wp.End) {
					t.Errorf("events[%d].Payload.End: want %v, got %v", i, wp.End, p.End)
				}
				if p.Attendee != wp.Attendee {
					t.Errorf("events[%d].Payload.Attendee: want %+v, got %+v", i, wp.Attendee, p.Attendee)
				}
			}
		})
	}
}
