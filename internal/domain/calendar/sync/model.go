package calendarsync

import (
	"encoding/json"
	"errors"
	"fmt"
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

func (s *Sync) MarshalJSON() ([]byte, error) {
	raw := struct {
		Email        string `json:"email"`
		CalendarID   string `json:"calendar_id"`
		SyncToken    string `json:"sync_token"`
		LastSyncedAt string `json:"last_synced_at"`
	}{
		Email:      s.email,
		CalendarID: s.calendarID,
		SyncToken:  s.syncToken,
	}
	if !s.lastSyncedAt.IsZero() {
		raw.LastSyncedAt = s.lastSyncedAt.UTC().Format(time.RFC3339)
	}
	return json.Marshal(raw)
}

// called by json.Unmarshal
func (s *Sync) UnmarshalJSON(data []byte) error {
	var raw struct {
		Email        string `json:"email"`
		CalendarID   string `json:"calendar_id"`
		SyncToken    string `json:"sync_token"`
		LastSyncedAt string `json:"last_synced_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.LastSyncedAt != "" {
		t, err := time.Parse(time.RFC3339, raw.LastSyncedAt)
		if err != nil {
			return fmt.Errorf("parse last_synced_at: %w", err)
		}
		s.lastSyncedAt = t
	}
	s.email = raw.Email
	s.calendarID = raw.CalendarID
	s.syncToken = raw.SyncToken
	return nil
}
