package calendarsync

import (
	"context"
	"fmt"

	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
)

type messageDispatcher interface {
	Dispatch(ctx context.Context, syncs []*domain.Sync) error
}

type syncStateRepository interface {
	FindAll(ctx context.Context) ([]*domain.Sync, error)
}

type CalendarSyncDispatcher struct {
	syncStates syncStateRepository
	dispatcher messageDispatcher
}

func NewCalendarSyncDispatcher(syncStates syncStateRepository, dispatcher messageDispatcher) *CalendarSyncDispatcher {
	return &CalendarSyncDispatcher{syncStates: syncStates, dispatcher: dispatcher}
}

func (d *CalendarSyncDispatcher) Handle(ctx context.Context) error {
	syncs, err := d.syncStates.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("find all sync states: %w", err)
	}
	if len(syncs) == 0 {
		return nil
	}
	return d.dispatcher.Dispatch(ctx, syncs)
}
