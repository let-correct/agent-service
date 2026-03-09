package calendarsync

import (
	"context"
	"errors"
	"time"

	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
)

const eventBuffer = 24 * time.Hour

type SyncCommand struct {
	Email        string
	CalendarID   string
	SyncToken    string
	LastSyncedAt time.Time
	AccessToken  string
}

type calendarClient interface {
	SyncEvents(ctx context.Context, email, accessToken, calendarID, syncToken string, lastSyncedAt time.Time) ([]domain.Event, string, error)
}

type eventPublisher interface {
	Publish(ctx context.Context, events []domain.Event) error
}

type syncRepository interface {
	Save(ctx context.Context, sync *domain.Sync) error
}

type SyncCalendar struct {
	client    calendarClient
	publisher eventPublisher
	syncs     syncRepository
	now       func() time.Time
}

func NewSyncCalendar(client calendarClient, publisher eventPublisher, syncs syncRepository) *SyncCalendar {
	return &SyncCalendar{
		client:    client,
		publisher: publisher,
		syncs:     syncs,
		now:       time.Now,
	}
}

func (s *SyncCalendar) Handle(ctx context.Context, cmd SyncCommand) error {
	from := s.pullEventsFromTime()
	pullEventsFrom := cmd.LastSyncedAt
	if cmd.SyncToken == "" && pullEventsFrom.IsZero() {
		pullEventsFrom = from
	}

	events, newSyncToken, err := s.client.SyncEvents(ctx, cmd.Email, cmd.AccessToken, cmd.CalendarID, cmd.SyncToken, pullEventsFrom)
	if errors.Is(err, domain.ErrSyncTokenExpired) {
		// Sync token expired: fall back to a full re-sync from eventBuffer ago.
		events, newSyncToken, err = s.client.SyncEvents(ctx, cmd.Email, cmd.AccessToken, cmd.CalendarID, "", from)
	}
	if err != nil {
		return err
	}

	if len(events) > 0 {
		if err := s.publisher.Publish(ctx, events); err != nil {
			return err
		}
	}

	return s.syncs.Save(ctx, domain.NewSync(cmd.Email, cmd.CalendarID, newSyncToken, s.now().UTC()))
}

func (s *SyncCalendar) pullEventsFromTime() time.Time {
	return s.now().Add(-eventBuffer).UTC()
}
