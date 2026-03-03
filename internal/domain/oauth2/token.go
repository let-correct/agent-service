package oauth2

import "time"

type ProviderID string

const (
	ProviderGoogle ProviderID = "google"
	ProviderArthur ProviderID = "arthur"
)

type Token struct {
	email        string
	provider     ProviderID
	accessToken  string
	refreshToken string
	expiresAt    time.Time
}

func (t *Token) IsExpired() bool {
	return time.Now().After(t.expiresAt)
}

func (t *Token) NeedsRefresh() bool {
	return time.Now().After(t.expiresAt.Add(-5 * time.Minute)) // refresh 5 mins early
}
