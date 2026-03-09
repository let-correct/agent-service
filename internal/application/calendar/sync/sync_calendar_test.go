package calendarsync

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
)

// -- mocks --

type syncEventsCall struct {
	syncToken    string
	lastSyncedAt time.Time
}

type syncEventsResponse struct {
	events    []domain.Event
	syncToken string
	err       error
}

type mockCalendarClient struct {
	calls     []syncEventsCall
	responses []syncEventsResponse
}

func (m *mockCalendarClient) SyncEvents(_ context.Context, _, _, _, syncToken string, lastSyncedAt time.Time) ([]domain.Event, string, error) {
	m.calls = append(m.calls, syncEventsCall{syncToken: syncToken, lastSyncedAt: lastSyncedAt})
	i := len(m.calls) - 1
	if i >= len(m.responses) {
		return nil, "", nil
	}
	r := m.responses[i]
	return r.events, r.syncToken, r.err
}

type mockPublisher struct {
	calledWith []domain.Event
	err        error
}

func (m *mockPublisher) Publish(_ context.Context, events []domain.Event) error {
	m.calledWith = append(m.calledWith, events...)
	return m.err
}

type mockSyncRepository struct {
	saved *domain.Sync
	err   error
}

func (m *mockSyncRepository) Save(_ context.Context, sync *domain.Sync) error {
	m.saved = sync
	return m.err
}

// -- helpers --

func newTestUseCase(client *mockCalendarClient, pub *mockPublisher, repo *mockSyncRepository, now func() time.Time) *SyncCalendar {
	return &SyncCalendar{client: client, publisher: pub, syncs: repo, now: now}
}

func baseCmd() SyncCommand {
	return SyncCommand{
		Email:       "agent@example.com",
		CalendarID:  "cal123",
		AccessToken: "access-tok",
	}
}

// -- tests --

func TestSyncCalendar_Handle(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	fixedNow := func() time.Time { return fixedTime }

	anEvent := domain.Event{DetailType: domain.DetailTypeCreated}

	tests := []struct {
		name              string
		cmd               SyncCommand
		clientResponses   []syncEventsResponse
		publishErr        error
		saveErr           error
		wantErr           error
		wantClientCalls   []syncEventsCall
		wantPublishedCount int
		wantSavedToken    string
		wantSavedAt       time.Time
	}{
		{
			name: "happy path: syncs, publishes, and saves",
			cmd:  baseCmd(),
			clientResponses: []syncEventsResponse{
				{events: []domain.Event{anEvent}, syncToken: "new-tok"},
			},
			wantClientCalls: []syncEventsCall{
				{syncToken: "", lastSyncedAt: fixedTime.Add(-eventBuffer).UTC()},
			},
			wantPublishedCount: 1,
			wantSavedToken:     "new-tok",
			wantSavedAt:        fixedTime.UTC(),
		},
		{
			name: "with syncToken: passes it through and uses lastSyncedAt as timeMin",
			cmd: SyncCommand{
				Email:        "agent@example.com",
				CalendarID:   "cal123",
				AccessToken:  "access-tok",
				SyncToken:    "existing-tok",
				LastSyncedAt: fixedTime.Add(-48 * time.Hour),
			},
			clientResponses: []syncEventsResponse{
				{syncToken: "updated-tok"},
			},
			wantClientCalls: []syncEventsCall{
				{syncToken: "existing-tok", lastSyncedAt: fixedTime.Add(-48 * time.Hour)},
			},
			wantSavedToken: "updated-tok",
			wantSavedAt:    fixedTime.UTC(),
		},
		{
			name: "with lastSyncedAt and no syncToken: uses lastSyncedAt as timeMin",
			cmd: SyncCommand{
				Email:        "agent@example.com",
				CalendarID:   "cal123",
				AccessToken:  "access-tok",
				LastSyncedAt: fixedTime.Add(-48 * time.Hour),
			},
			clientResponses: []syncEventsResponse{
				{syncToken: "new-tok"},
			},
			wantClientCalls: []syncEventsCall{
				{syncToken: "", lastSyncedAt: fixedTime.Add(-48 * time.Hour)},
			},
			wantSavedToken: "new-tok",
			wantSavedAt:    fixedTime.UTC(),
		},
		{
			name: "no syncToken and no lastSyncedAt: uses now minus eventBuffer",
			cmd:  baseCmd(),
			clientResponses: []syncEventsResponse{
				{syncToken: "new-tok"},
			},
			wantClientCalls: []syncEventsCall{
				{syncToken: "", lastSyncedAt: fixedTime.Add(-eventBuffer).UTC()},
			},
			wantSavedToken: "new-tok",
			wantSavedAt:    fixedTime.UTC(),
		},
		{
			name: "expired sync token: retries with empty token and now minus eventBuffer",
			cmd: SyncCommand{
				Email:       "agent@example.com",
				CalendarID:  "cal123",
				AccessToken: "access-tok",
				SyncToken:   "stale-tok",
			},
			clientResponses: []syncEventsResponse{
				{err: domain.ErrSyncTokenExpired},
				{events: []domain.Event{anEvent}, syncToken: "fresh-tok"},
			},
			wantClientCalls: []syncEventsCall{
				{syncToken: "stale-tok", lastSyncedAt: time.Time{}},
				{syncToken: "", lastSyncedAt: fixedTime.Add(-eventBuffer).UTC()},
			},
			wantPublishedCount: 1,
			wantSavedToken:     "fresh-tok",
			wantSavedAt:        fixedTime.UTC(),
		},
		{
			name: "no events: skips publish but still saves",
			cmd:  baseCmd(),
			clientResponses: []syncEventsResponse{
				{syncToken: "new-tok"},
			},
			wantPublishedCount: 0,
			wantSavedToken:     "new-tok",
			wantSavedAt:        fixedTime.UTC(),
		},
		{
			name:    "SyncEvents error is returned",
			cmd:     baseCmd(),
			clientResponses: []syncEventsResponse{
				{err: errors.New("api error")},
			},
			wantErr: errors.New("api error"),
		},
		{
			name: "expired token retry error is returned",
			cmd: SyncCommand{
				Email:       "agent@example.com",
				CalendarID:  "cal123",
				AccessToken: "access-tok",
				SyncToken:   "stale-tok",
			},
			clientResponses: []syncEventsResponse{
				{err: domain.ErrSyncTokenExpired},
				{err: errors.New("retry failed")},
			},
			wantErr: errors.New("retry failed"),
		},
		{
			name: "publisher error is returned",
			cmd:  baseCmd(),
			clientResponses: []syncEventsResponse{
				{events: []domain.Event{anEvent}},
			},
			publishErr: errors.New("publish failed"),
			wantErr:    errors.New("publish failed"),
		},
		{
			name: "save error is returned",
			cmd:  baseCmd(),
			clientResponses: []syncEventsResponse{
				{syncToken: "new-tok"},
			},
			saveErr: errors.New("save failed"),
			wantErr: errors.New("save failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockCalendarClient{responses: tt.clientResponses}
			pub := &mockPublisher{err: tt.publishErr}
			repo := &mockSyncRepository{err: tt.saveErr}
			uc := newTestUseCase(client, pub, repo, fixedNow)

			err := uc.Handle(context.Background(), tt.cmd)

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

			if len(tt.wantClientCalls) > 0 {
				if len(client.calls) != len(tt.wantClientCalls) {
					t.Fatalf("client calls: want %d, got %d", len(tt.wantClientCalls), len(client.calls))
				}
				for i, want := range tt.wantClientCalls {
					got := client.calls[i]
					if got.syncToken != want.syncToken {
						t.Errorf("call[%d].syncToken: want %q, got %q", i, want.syncToken, got.syncToken)
					}
					if !got.lastSyncedAt.Equal(want.lastSyncedAt) {
						t.Errorf("call[%d].lastSyncedAt: want %v, got %v", i, want.lastSyncedAt, got.lastSyncedAt)
					}
				}
			}

			if len(pub.calledWith) != tt.wantPublishedCount {
				t.Errorf("published events: want %d, got %d", tt.wantPublishedCount, len(pub.calledWith))
			}

			if repo.saved == nil {
				t.Fatal("expected sync to be saved, but Save was not called")
			}
			if repo.saved.SyncToken() != tt.wantSavedToken {
				t.Errorf("saved syncToken: want %q, got %q", tt.wantSavedToken, repo.saved.SyncToken())
			}
			if !repo.saved.LastSyncedAt().Equal(tt.wantSavedAt) {
				t.Errorf("saved lastSyncedAt: want %v, got %v", tt.wantSavedAt, repo.saved.LastSyncedAt())
			}
		})
	}
}
