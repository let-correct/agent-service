package calendarworker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	calendarsync "github.com/troysnowden/agent-service/internal/application/calendar/sync"
)

type mockCalendarSyncer struct {
	calledWith []calendarsync.SyncCommand
	err        error
}

func (m *mockCalendarSyncer) Handle(_ context.Context, cmd calendarsync.SyncCommand) error {
	m.calledWith = append(m.calledWith, cmd)
	return m.err
}

func newLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func sqsRecord(messageID, body string) events.SQSMessage {
	return events.SQSMessage{MessageId: messageID, Body: body}
}

func TestHandle(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name              string
		records           []events.SQSMessage
		syncerErr         error
		wantBatchFailures []string
		wantSyncCommands  []calendarsync.SyncCommand
	}{
		{
			name:    "empty batch returns no failures",
			records: []events.SQSMessage{},
		},
		{
			name: "single valid record maps all fields to sync command",
			records: []events.SQSMessage{
				sqsRecord("msg-1", `{"email":"agent@example.com","calendar_id":"cal123","sync_token":"tok-abc","last_synced_at":"2024-01-15T12:00:00Z"}`),
			},
			wantSyncCommands: []calendarsync.SyncCommand{
				{
					Email:        "agent@example.com",
					CalendarID:   "cal123",
					SyncToken:    "tok-abc",
					LastSyncedAt: fixedTime,
				},
			},
		},
		{
			name: "empty last_synced_at maps to zero time",
			records: []events.SQSMessage{
				sqsRecord("msg-1", `{"email":"agent@example.com","calendar_id":"cal123"}`),
			},
			wantSyncCommands: []calendarsync.SyncCommand{
				{Email: "agent@example.com", CalendarID: "cal123"},
			},
		},
		{
			name: "invalid JSON body reports batch item failure",
			records: []events.SQSMessage{
				sqsRecord("msg-1", `not-json`),
			},
			wantBatchFailures: []string{"msg-1"},
		},
		{
			name: "invalid last_synced_at timestamp reports batch item failure",
			records: []events.SQSMessage{
				sqsRecord("msg-1", `{"email":"agent@example.com","last_synced_at":"not-a-date"}`),
			},
			wantBatchFailures: []string{"msg-1"},
		},
		{
			name: "syncer error reports batch item failure",
			records: []events.SQSMessage{
				sqsRecord("msg-1", `{"email":"agent@example.com","calendar_id":"cal123"}`),
			},
			syncerErr:         errors.New("sync failed"),
			wantBatchFailures: []string{"msg-1"},
			wantSyncCommands: []calendarsync.SyncCommand{
				{Email: "agent@example.com", CalendarID: "cal123"},
			},
		},
		{
			name: "partial failure reports only failed record",
			records: []events.SQSMessage{
				sqsRecord("msg-1", `{"email":"a@example.com","calendar_id":"cal-a"}`),
				sqsRecord("msg-2", `not-json`),
				sqsRecord("msg-3", `{"email":"c@example.com","calendar_id":"cal-c"}`),
			},
			wantBatchFailures: []string{"msg-2"},
			wantSyncCommands: []calendarsync.SyncCommand{
				{Email: "a@example.com", CalendarID: "cal-a"},
				{Email: "c@example.com", CalendarID: "cal-c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncer := &mockCalendarSyncer{err: tt.syncerErr}
			h := New(newLogger(), syncer)

			resp, err := h.Handle(context.Background(), events.SQSEvent{Records: tt.records})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resp.BatchItemFailures) != len(tt.wantBatchFailures) {
				t.Fatalf("batch failures: want %d, got %d", len(tt.wantBatchFailures), len(resp.BatchItemFailures))
			}
			for i, id := range tt.wantBatchFailures {
				if got := resp.BatchItemFailures[i].ItemIdentifier; got != id {
					t.Errorf("BatchItemFailures[%d]: want %q, got %q", i, id, got)
				}
			}

			if len(syncer.calledWith) != len(tt.wantSyncCommands) {
				t.Fatalf("syncer calls: want %d, got %d", len(tt.wantSyncCommands), len(syncer.calledWith))
			}
			for i, want := range tt.wantSyncCommands {
				got := syncer.calledWith[i]
				if got.Email != want.Email {
					t.Errorf("SyncCommand[%d].Email: want %q, got %q", i, want.Email, got.Email)
				}
				if got.CalendarID != want.CalendarID {
					t.Errorf("SyncCommand[%d].CalendarID: want %q, got %q", i, want.CalendarID, got.CalendarID)
				}
				if got.SyncToken != want.SyncToken {
					t.Errorf("SyncCommand[%d].SyncToken: want %q, got %q", i, want.SyncToken, got.SyncToken)
				}
				if !got.LastSyncedAt.Equal(want.LastSyncedAt) {
					t.Errorf("SyncCommand[%d].LastSyncedAt: want %v, got %v", i, want.LastSyncedAt, got.LastSyncedAt)
				}
			}
		})
	}
}
