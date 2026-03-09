package calendarsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
	"github.com/troysnowden/agent-service/internal/domain/oauth2"
)

const eventBuffer = 24 * time.Hour

type SyncCommand struct {
	Email        string
	CalendarID   string
	SyncToken    string
	LastSyncedAt time.Time
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

type tokenRepository interface {
	FindByEmail(ctx context.Context, email string) (*oauth2.Token, error)
}

type SyncCalendar struct {
	client    calendarClient
	publisher eventPublisher
	syncs     syncRepository
	tokens    tokenRepository
	now       func() time.Time
}

func NewSyncCalendar(client calendarClient, publisher eventPublisher, syncs syncRepository, tokens tokenRepository) *SyncCalendar {
	return &SyncCalendar{
		client:    client,
		publisher: publisher,
		syncs:     syncs,
		tokens:    tokens,
		now:       time.Now,
	}
}

func (s *SyncCalendar) Handle(ctx context.Context, cmd SyncCommand) error {
	token, err := s.tokens.FindByEmail(ctx, cmd.Email)
	if err != nil {
		return fmt.Errorf("find token: %w", err)
	}

	from := s.pullEventsFromTime()
	pullEventsFrom := cmd.LastSyncedAt
	if cmd.SyncToken == "" && pullEventsFrom.IsZero() {
		pullEventsFrom = from
	}

	events, newSyncToken, err := s.client.SyncEvents(ctx, cmd.Email, token.AccessToken(), cmd.CalendarID, cmd.SyncToken, pullEventsFrom)
	if errors.Is(err, domain.ErrSyncTokenExpired) {
		// Sync token expired: fall back to a full re-sync from eventBuffer ago.
		events, newSyncToken, err = s.client.SyncEvents(ctx, cmd.Email, token.AccessToken(), cmd.CalendarID, "", from)
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
