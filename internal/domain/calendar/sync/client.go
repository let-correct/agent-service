package calendarsync

import (
	"context"
	"time"
)

type Client interface {
	SyncEvents(ctx context.Context, email, accessToken, calendarID, syncToken string, lastSyncedAt time.Time) ([]Event, string, error)
}
