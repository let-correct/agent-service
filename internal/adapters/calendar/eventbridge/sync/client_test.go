package calendarsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
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
			AgentEmail: "user@example.com",
			Location:   "10 Main St, Dublin",
			Attendee: domain.Attendee{
				Name:  "John Smith",
				Email: "john@example.com",
				Phone: "0821234567",
			},
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
			name:   "single event is published successfully with full event as detail",
			events: []domain.Event{sampleEvent(domain.DetailTypeCreated, source, "evt1")},
			setupMock: func(m *mockEventbridgeClient) {
				m.On("PutEvents", mock.Anything, mock.MatchedBy(func(input *eventbridge.PutEventsInput) bool {
					if len(input.Entries) != 1 {
						return false
					}
					entry := input.Entries[0]
					if *entry.DetailType != string(domain.DetailTypeCreated) {
						return false
					}
					if *entry.EventBusName != busName {
						return false
					}
					if *entry.Source != source {
						return false
					}
					var published domain.Event
					if err := json.Unmarshal([]byte(*entry.Detail), &published); err != nil {
						return false
					}
					return published.Metadata.CorrelationID == "evt1" &&
						published.Metadata.Source == source &&
						published.Payload.AgentEmail == "user@example.com"
				})).Return(&eventbridge.PutEventsOutput{
					Entries: []ebtypes.PutEventsResultEntry{{EventId: aws.String("id-1")}},
				}, nil)
			},
		},
		{
			name: "events within batch size are sent in a single PutEvents call",
			events: []domain.Event{
				sampleEvent(domain.DetailTypeCreated, source, "evt1"),
				sampleEvent(domain.DetailTypeCancelled, source, "evt2"),
			},
			setupMock: func(m *mockEventbridgeClient) {
				m.On("PutEvents", mock.Anything, mock.MatchedBy(func(input *eventbridge.PutEventsInput) bool {
					return len(input.Entries) == 2
				})).Return(&eventbridge.PutEventsOutput{
					Entries: []ebtypes.PutEventsResultEntry{
						{EventId: aws.String("id-1")},
						{EventId: aws.String("id-2")},
					},
				}, nil)
			},
		},
		{
			name: "events exceeding batch size are chunked into multiple PutEvents calls",
			events: func() []domain.Event {
				evts := make([]domain.Event, 11)
				for i := range evts {
					evts[i] = sampleEvent(domain.DetailTypeCreated, source, fmt.Sprintf("evt%d", i+1))
				}
				return evts
			}(),
			setupMock: func(m *mockEventbridgeClient) {
				successEntries := func(n int) []ebtypes.PutEventsResultEntry {
					out := make([]ebtypes.PutEventsResultEntry, n)
					for i := range out {
						out[i] = ebtypes.PutEventsResultEntry{EventId: aws.String(fmt.Sprintf("id-%d", i+1))}
					}
					return out
				}
				m.On("PutEvents", mock.Anything, mock.MatchedBy(func(input *eventbridge.PutEventsInput) bool {
					return len(input.Entries) == 10
				})).Return(&eventbridge.PutEventsOutput{Entries: successEntries(10)}, nil).Once()
				m.On("PutEvents", mock.Anything, mock.MatchedBy(func(input *eventbridge.PutEventsInput) bool {
					return len(input.Entries) == 1
				})).Return(&eventbridge.PutEventsOutput{Entries: successEntries(1)}, nil).Once()
			},
		},
		{
			name: "failed entries that persist beyond max retries return an error",
			events: []domain.Event{
				sampleEvent(domain.DetailTypeCreated, source, "evt1"),
				sampleEvent(domain.DetailTypeCreated, source, "evt2"),
			},
			setupMock: func(m *mockEventbridgeClient) {
				persistentFailure := &eventbridge.PutEventsOutput{
					FailedEntryCount: 1,
					Entries: []ebtypes.PutEventsResultEntry{
						{EventId: aws.String("id-1")},
						{ErrorCode: aws.String("ThrottlingException"), ErrorMessage: aws.String("rate exceeded")},
					},
				}
				// All 3 attempts fail on the second entry.
				m.On("PutEvents", mock.Anything, mock.MatchedBy(func(input *eventbridge.PutEventsInput) bool {
					return len(input.Entries) == 2
				})).Return(persistentFailure, nil).Once()
				m.On("PutEvents", mock.Anything, mock.MatchedBy(func(input *eventbridge.PutEventsInput) bool {
					return len(input.Entries) == 1
				})).Return(&eventbridge.PutEventsOutput{
					FailedEntryCount: 1,
					Entries: []ebtypes.PutEventsResultEntry{
						{ErrorCode: aws.String("ThrottlingException"), ErrorMessage: aws.String("rate exceeded")},
					},
				}, nil).Times(2)
			},
			wantErr:       true,
			wantErrSubstr: "1 events failed after 3 attempts",
		},
		{
			name: "failed entries within a batch are retried until they succeed",
			events: []domain.Event{
				sampleEvent(domain.DetailTypeCreated, source, "evt1"),
				sampleEvent(domain.DetailTypeCreated, source, "evt2"),
				sampleEvent(domain.DetailTypeCreated, source, "evt3"),
			},
			setupMock: func(m *mockEventbridgeClient) {
				// First attempt: middle entry fails.
				m.On("PutEvents", mock.Anything, mock.MatchedBy(func(input *eventbridge.PutEventsInput) bool {
					return len(input.Entries) == 3
				})).Return(&eventbridge.PutEventsOutput{
					FailedEntryCount: 1,
					Entries: []ebtypes.PutEventsResultEntry{
						{EventId: aws.String("id-1")},
						{ErrorCode: aws.String("ThrottlingException"), ErrorMessage: aws.String("rate exceeded")},
						{EventId: aws.String("id-3")},
					},
				}, nil).Once()
				// Retry: only the one failed entry, succeeds.
				m.On("PutEvents", mock.Anything, mock.MatchedBy(func(input *eventbridge.PutEventsInput) bool {
					return len(input.Entries) == 1
				})).Return(&eventbridge.PutEventsOutput{
					Entries: []ebtypes.PutEventsResultEntry{{EventId: aws.String("id-2")}},
				}, nil).Once()
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
