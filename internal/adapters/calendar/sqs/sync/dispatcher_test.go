package calendarsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
)

type mockSQSClient struct {
	mock.Mock
}

func (m *mockSQSClient) SendMessageBatch(ctx context.Context, params *sqs.SendMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageBatchOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.SendMessageBatchOutput), args.Error(1)
}

func newTestDispatcher(client *mockSQSClient, queueURL string) *Dispatcher {
	return &Dispatcher{client: client, queueURL: queueURL}
}

func newSync(email, calendarID, syncToken string, lastSyncedAt time.Time) *domain.Sync {
	return domain.NewSync(email, calendarID, syncToken, lastSyncedAt)
}

func successOutput(n int) *sqs.SendMessageBatchOutput {
	entries := make([]sqstypes.SendMessageBatchResultEntry, n)
	for i := range entries {
		id := fmt.Sprintf("%d", i)
		entries[i] = sqstypes.SendMessageBatchResultEntry{Id: &id}
	}
	return &sqs.SendMessageBatchOutput{Successful: entries}
}

func TestDispatch(t *testing.T) {
	const queueURL = "https://sqs.us-east-1.amazonaws.com/123/calendar-sync"

	fixedTime := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		syncs         []*domain.Sync
		setupMock     func(*mockSQSClient)
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:      "empty syncs skips SendMessageBatch",
			syncs:     []*domain.Sync{},
			setupMock: func(m *mockSQSClient) {},
		},
		{
			name:  "nil syncs skips SendMessageBatch",
			syncs: nil,
			setupMock: func(m *mockSQSClient) {},
		},
		{
			name:  "single sync is dispatched successfully",
			syncs: []*domain.Sync{newSync("agent@example.com", "primary", "token123", fixedTime)},
			setupMock: func(m *mockSQSClient) {
				m.On("SendMessageBatch", mock.Anything, mock.MatchedBy(func(input *sqs.SendMessageBatchInput) bool {
					return *input.QueueUrl == queueURL && len(input.Entries) == 1
				})).Return(successOutput(1), nil)
			},
		},
		{
			name: "message body is correctly serialised",
			syncs: []*domain.Sync{newSync("agent@example.com", "primary", "tok", fixedTime)},
			setupMock: func(m *mockSQSClient) {
				m.On("SendMessageBatch", mock.Anything, mock.MatchedBy(func(input *sqs.SendMessageBatchInput) bool {
					var payload map[string]string
					if err := json.Unmarshal([]byte(*input.Entries[0].MessageBody), &payload); err != nil {
						return false
					}
					return payload["email"] == "agent@example.com" &&
						payload["calendar_id"] == "primary" &&
						payload["sync_token"] == "tok" &&
						payload["last_synced_at"] == fixedTime.UTC().Format(time.RFC3339)
				})).Return(successOutput(1), nil)
			},
		},
		{
			name: "zero last_synced_at is serialised as empty string",
			syncs: []*domain.Sync{newSync("agent@example.com", "primary", "", time.Time{})},
			setupMock: func(m *mockSQSClient) {
				m.On("SendMessageBatch", mock.Anything, mock.MatchedBy(func(input *sqs.SendMessageBatchInput) bool {
					var payload map[string]string
					if err := json.Unmarshal([]byte(*input.Entries[0].MessageBody), &payload); err != nil {
						return false
					}
					return payload["last_synced_at"] == ""
				})).Return(successOutput(1), nil)
			},
		},
		{
			name: "syncs within batch size sent in a single call",
			syncs: func() []*domain.Sync {
				out := make([]*domain.Sync, 5)
				for i := range out {
					out[i] = newSync(fmt.Sprintf("agent%d@example.com", i), "primary", "", fixedTime)
				}
				return out
			}(),
			setupMock: func(m *mockSQSClient) {
				m.On("SendMessageBatch", mock.Anything, mock.MatchedBy(func(input *sqs.SendMessageBatchInput) bool {
					return len(input.Entries) == 5
				})).Return(successOutput(5), nil).Once()
			},
		},
		{
			name: "syncs exceeding batch size are chunked into multiple calls",
			syncs: func() []*domain.Sync {
				out := make([]*domain.Sync, 11)
				for i := range out {
					out[i] = newSync(fmt.Sprintf("agent%d@example.com", i), "primary", "", fixedTime)
				}
				return out
			}(),
			setupMock: func(m *mockSQSClient) {
				m.On("SendMessageBatch", mock.Anything, mock.MatchedBy(func(input *sqs.SendMessageBatchInput) bool {
					return len(input.Entries) == 10
				})).Return(successOutput(10), nil).Once()
				m.On("SendMessageBatch", mock.Anything, mock.MatchedBy(func(input *sqs.SendMessageBatchInput) bool {
					return len(input.Entries) == 1
				})).Return(successOutput(1), nil).Once()
			},
		},
		{
			name: "message IDs are globally unique across batches",
			syncs: func() []*domain.Sync {
				out := make([]*domain.Sync, 11)
				for i := range out {
					out[i] = newSync(fmt.Sprintf("agent%d@example.com", i), "primary", "", fixedTime)
				}
				return out
			}(),
			setupMock: func(m *mockSQSClient) {
				seen := map[string]bool{}
				m.On("SendMessageBatch", mock.Anything, mock.MatchedBy(func(input *sqs.SendMessageBatchInput) bool {
					for _, e := range input.Entries {
						if seen[*e.Id] {
							return false
						}
						seen[*e.Id] = true
					}
					return true
				})).Return(successOutput(len(seen)), nil).Times(2)
			},
		},
		{
			name:  "SDK error is propagated",
			syncs: []*domain.Sync{newSync("agent@example.com", "primary", "", fixedTime)},
			setupMock: func(m *mockSQSClient) {
				m.On("SendMessageBatch", mock.Anything, mock.Anything).
					Return(nil, errors.New("connection refused"))
			},
			wantErr:       true,
			wantErrSubstr: "sqs send message batch",
		},
		{
			name:  "partial batch failure returns error",
			syncs: []*domain.Sync{newSync("a@example.com", "primary", "", fixedTime), newSync("b@example.com", "primary", "", fixedTime)},
			setupMock: func(m *mockSQSClient) {
				m.On("SendMessageBatch", mock.Anything, mock.Anything).
					Return(&sqs.SendMessageBatchOutput{
						Successful: []sqstypes.SendMessageBatchResultEntry{{Id: func() *string { s := "0"; return &s }()}},
						Failed:     []sqstypes.BatchResultErrorEntry{{Id: func() *string { s := "1"; return &s }(), Code: func() *string { s := "ThrottlingException"; return &s }()}},
					}, nil)
			},
			wantErr:       true,
			wantErrSubstr: "1 messages failed in batch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockSQSClient{}
			tt.setupMock(m)

			d := newTestDispatcher(m, queueURL)
			err := d.Dispatch(context.Background(), tt.syncs)

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
