package oauth2

import (
	"context"
	"errors"
	"time"
)

var ErrTokenNotFound = errors.New("token not found")

type TokenRepository interface {
	Save(ctx context.Context, token *Token) error
	FindByEmailAndProvider(ctx context.Context, email string, provider ProviderID) (*Token, error)
}

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

func NewToken(email string, provider ProviderID, accessToken, refreshToken string, expiresAt time.Time) *Token {
	return &Token{
		email:        email,
		provider:     provider,
		accessToken:  accessToken,
		refreshToken: refreshToken,
		expiresAt:    expiresAt,
	}
}

func (t *Token) Email() string        { return t.email }
func (t *Token) Provider() ProviderID { return t.provider }
func (t *Token) AccessToken() string  { return t.accessToken }
func (t *Token) RefreshToken() string { return t.refreshToken }
func (t *Token) ExpiresAt() time.Time { return t.expiresAt }

func (t *Token) IsExpired() bool {
	return time.Now().After(t.expiresAt)
}

func (t *Token) NeedsRefresh() bool {
	return time.Now().After(t.expiresAt.Add(-5 * time.Minute)) // refresh 5 mins early
}
