package calendarsync

import (
	"errors"
	"time"
)

var ErrSyncTokenExpired = errors.New("sync token expired")

type Sync struct {
	email        string
	calendarID   string
	syncToken    string // empty = full sync needed
	lastSyncedAt time.Time
}

func NewSync(email, calendarID, syncToken string, lastSyncedAt time.Time) *Sync {
	return &Sync{
		email:        email,
		calendarID:   calendarID,
		syncToken:    syncToken,
		lastSyncedAt: lastSyncedAt,
	}
}

func (s *Sync) Email() string           { return s.email }
func (s *Sync) CalendarID() string      { return s.calendarID }
func (s *Sync) SyncToken() string       { return s.syncToken }
func (s *Sync) LastSyncedAt() time.Time { return s.lastSyncedAt }
