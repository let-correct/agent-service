package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/troysnowden/agent-service/internal/domain/oauth2"
)

type mockGetTokenRepo struct {
	findToken *oauth2.Token
	findErr   error
	saved     bool
	saveErr   error
}

func (m *mockGetTokenRepo) Save(_ context.Context, _ *oauth2.Token) error {
	m.saved = true
	return m.saveErr
}

func (m *mockGetTokenRepo) FindByEmailAndProvider(_ context.Context, _ string, _ oauth2.ProviderID) (*oauth2.Token, error) {
	return m.findToken, m.findErr
}

type mockGetTokenClient struct {
	refreshCalled bool
	refreshResult *oauth2.Token
	refreshErr    error
}

func (m *mockGetTokenClient) AuthorizationURL(_ context.Context, _ string) string { return "" }
func (m *mockGetTokenClient) ExchangeCode(_ context.Context, _, _ string) (*oauth2.Token, error) {
	return nil, nil
}
func (m *mockGetTokenClient) RefreshToken(_ context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	m.refreshCalled = true
	if m.refreshErr != nil {
		return nil, m.refreshErr
	}
	if m.refreshResult != nil {
		return m.refreshResult, nil
	}
	return token, nil
}

func TestGetToken_Handle(t *testing.T) {
	now := time.Now()
	freshToken := oauth2.NewToken("user@example.com", oauth2.ProviderGoogle, "fresh-access", "refresh", now.Add(time.Hour))
	nearExpiryToken := oauth2.NewToken("user@example.com", oauth2.ProviderGoogle, "stale-access", "refresh", now.Add(2*time.Minute))
	expiredToken := oauth2.NewToken("user@example.com", oauth2.ProviderGoogle, "old-access", "refresh", now.Add(-time.Hour))
	refreshedToken := oauth2.NewToken("user@example.com", oauth2.ProviderGoogle, "new-access", "new-refresh", now.Add(time.Hour))

	tests := []struct {
		name     string
		provider oauth2.ProviderID

		findToken      *oauth2.Token
		findErr        error
		refreshResult  *oauth2.Token
		refreshErr     error
		saveErr        error

		wantErr           error
		wantAccessToken   string
		wantRefreshCalled bool
		wantSaved         bool
	}{
		{
			name:            "fresh token returned without refresh",
			provider:        oauth2.ProviderGoogle,
			findToken:       freshToken,
			wantAccessToken: "fresh-access",
		},
		{
			name:              "near-expiry token is refreshed, saved, and returned",
			provider:          oauth2.ProviderGoogle,
			findToken:         nearExpiryToken,
			refreshResult:     refreshedToken,
			wantRefreshCalled: true,
			wantSaved:         true,
			wantAccessToken:   "new-access",
		},
		{
			name:              "expired token is refreshed and returned",
			provider:          oauth2.ProviderGoogle,
			findToken:         expiredToken,
			refreshResult:     refreshedToken,
			wantRefreshCalled: true,
			wantSaved:         true,
			wantAccessToken:   "new-access",
		},
		{
			name:     "unknown provider returns ErrUnsupportedProvider without I/O",
			provider: oauth2.ProviderID("unknown"),
			wantErr:  ErrUnsupportedProvider,
		},
		{
			name:     "token not found returns ErrTokenNotFound",
			provider: oauth2.ProviderGoogle,
			findErr:  oauth2.ErrTokenNotFound,
			wantErr:  oauth2.ErrTokenNotFound,
		},
		{
			name:     "repo find failure is propagated",
			provider: oauth2.ProviderGoogle,
			findErr:  errors.New("dynamodb unavailable"),
			wantErr:  errors.New("dynamodb unavailable"),
		},
		{
			name:              "refresh failure is returned without saving",
			provider:          oauth2.ProviderGoogle,
			findToken:         nearExpiryToken,
			refreshErr:        errors.New("provider rejected refresh"),
			wantErr:           errors.New("provider rejected refresh"),
			wantRefreshCalled: true,
		},
		{
			name:              "save failure after refresh is returned",
			provider:          oauth2.ProviderGoogle,
			findToken:         nearExpiryToken,
			refreshResult:     refreshedToken,
			saveErr:           errors.New("dynamodb write failed"),
			wantErr:           errors.New("dynamodb write failed"),
			wantRefreshCalled: true,
			wantSaved:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockGetTokenClient{refreshResult: tt.refreshResult, refreshErr: tt.refreshErr}
			tokens := &mockGetTokenRepo{findToken: tt.findToken, findErr: tt.findErr, saveErr: tt.saveErr}
			clients := map[oauth2.ProviderID]oauth2.Client{oauth2.ProviderGoogle: client}

			h := NewGetToken(clients, tokens)
			result, err := h.Handle(context.Background(), NewGetTokenCommand("user@example.com", tt.provider))

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.AccessToken != tt.wantAccessToken {
					t.Errorf("access token = %q, want %q", result.AccessToken, tt.wantAccessToken)
				}
				if result.ExpiresAt.IsZero() {
					t.Error("ExpiresAt should not be zero")
				}
			}

			if tt.wantRefreshCalled && !client.refreshCalled {
				t.Error("expected RefreshToken to be called, but it was not")
			}
			if !tt.wantRefreshCalled && client.refreshCalled {
				t.Error("RefreshToken should not have been called")
			}
			if tt.wantSaved && !tokens.saved {
				t.Error("expected token to be saved after refresh, but it was not")
			}
			if !tt.wantSaved && tokens.saved {
				t.Error("token should not have been saved")
			}
		})
	}
}
