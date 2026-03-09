package calendarsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
)

// mockEventbridgeAPI implements eventbridgeAPI for testing.
type mockEventbridgeClient struct {
	mock.Mock
}

func (m *mockEventbridgeClient) PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*eventbridge.PutEventsOutput), args.Error(1)
}

func newTestEventbridgeClient(eb *mockEventbridgeClient, busName string) *Eventbridge {
	return &Eventbridge{
		eventbridge: eb,
		busName:     busName,
	}
}

func sampleEvent(detailType domain.DetailType, source, correlationID string) domain.Event {
	return domain.Event{
		DetailType: detailType,
		Metadata: domain.EventMetadata{
			Timestamp:     time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			Source:        source,
			SchemaVersion: "1.0",
			CorrelationID: correlationID,
		},
		Payload: domain.Appointment{
			Email:      "user@example.com",
			EventID:    correlationID,
			CalendarID: "cal123",
			Summary:    "Standup",
			Status:     "confirmed",
			Attendees:  []string{"a@example.com", "b@example.com"},
		},
	}
}

func TestPublish(t *testing.T) {
	const busName = "my-event-bus"
	const source = "calendar-sync"

	tests := []struct {
		name          string
		events        []domain.Event
		setupMock     func(*mockEventbridgeClient)
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:      "empty events slice skips PutEvents",
			events:    []domain.Event{},
			setupMock: func(m *mockEventbridgeClient) {},
		},
		{
			name:   "PutEvents SDK error is propagated",
			events: []domain.Event{sampleEvent(domain.DetailTypeCreated, source, "evt1")},
			setupMock: func(m *mockEventbridgeClient) {
				m.On("PutEvents", mock.Anything, mock.Anything).
					Return(nil, errors.New("sdk error"))
			},
			wantErr:       true,
			wantErrSubstr: "put events",
		},
		{
			name:   "failed entries returns error",
			events: []domain.Event{sampleEvent(domain.DetailTypeCreated, source, "evt1")},
			setupMock: func(m *mockEventbridgeClient) {
				m.On("PutEvents", mock.Anything, mock.Anything).
					Return(&eventbridge.PutEventsOutput{FailedEntryCount: 1}, nil)
			},
			wantErr:       true,
			wantErrSubstr: "1 events failed to publish",
		},
		{
			name:   "single event is published successfully",
			events: []domain.Event{sampleEvent(domain.DetailTypeCreated, source, "evt1")},
			setupMock: func(m *mockEventbridgeClient) {
				m.On("PutEvents", mock.Anything, mock.MatchedBy(func(input *eventbridge.PutEventsInput) bool {
					return len(input.Entries) == 1 &&
						*input.Entries[0].DetailType == string(domain.DetailTypeCreated) &&
						*input.Entries[0].EventBusName == busName &&
						*input.Entries[0].Source == source
				})).Return(&eventbridge.PutEventsOutput{FailedEntryCount: 0}, nil)
			},
		},
		{
			name: "multiple events are sent in a single PutEvents call",
			events: []domain.Event{
				sampleEvent(domain.DetailTypeCreated, source, "evt1"),
				sampleEvent(domain.DetailTypeCancelled, source, "evt2"),
			},
			setupMock: func(m *mockEventbridgeClient) {
				m.On("PutEvents", mock.Anything, mock.MatchedBy(func(input *eventbridge.PutEventsInput) bool {
					return len(input.Entries) == 2
				})).Return(&eventbridge.PutEventsOutput{FailedEntryCount: 0}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockEventbridgeClient{}
			tt.setupMock(m)

			c := newTestEventbridgeClient(m, busName)
			err := c.Publish(context.Background(), tt.events)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}
