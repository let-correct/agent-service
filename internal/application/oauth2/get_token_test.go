package oauth2

import (
	"context"
	"errors"
	"testing"
	"time"

	oauth2domain "github.com/troysnowden/agent-service/internal/domain/oauth2"
)

func TestGetToken_Handle(t *testing.T) {
	now := time.Now()
	freshToken := oauth2domain.NewToken("user@example.com", oauth2domain.ProviderGoogle, "fresh-access", "refresh", now.Add(time.Hour))
	nearExpiryToken := oauth2domain.NewToken("user@example.com", oauth2domain.ProviderGoogle, "stale-access", "refresh", now.Add(2*time.Minute))
	expiredToken := oauth2domain.NewToken("user@example.com", oauth2domain.ProviderGoogle, "old-access", "refresh", now.Add(-time.Hour))
	refreshedToken := oauth2domain.NewToken("user@example.com", oauth2domain.ProviderGoogle, "new-access", "new-refresh", now.Add(time.Hour))

	tests := []struct {
		name     string
		provider oauth2domain.ProviderID

		findToken     *oauth2domain.Token
		findErr       error
		refreshResult *oauth2domain.Token
		refreshErr    error
		saveErr       error

		wantErr           error
		wantAccessToken   string
		wantRefreshCalled bool
		wantSaved         bool
	}{
		{
			name:            "fresh token returned without refresh",
			provider:        oauth2domain.ProviderGoogle,
			findToken:       freshToken,
			wantAccessToken: "fresh-access",
		},
		{
			name:              "near-expiry token is refreshed, saved, and returned",
			provider:          oauth2domain.ProviderGoogle,
			findToken:         nearExpiryToken,
			refreshResult:     refreshedToken,
			wantRefreshCalled: true,
			wantSaved:         true,
			wantAccessToken:   "new-access",
		},
		{
			name:              "expired token is refreshed and returned",
			provider:          oauth2domain.ProviderGoogle,
			findToken:         expiredToken,
			refreshResult:     refreshedToken,
			wantRefreshCalled: true,
			wantSaved:         true,
			wantAccessToken:   "new-access",
		},
		{
			name:     "unknown provider returns ErrUnsupportedProvider without I/O",
			provider: oauth2domain.ProviderID("unknown"),
			wantErr:  ErrUnsupportedProvider,
		},
		{
			name:     "token not found returns ErrTokenNotFound",
			provider: oauth2domain.ProviderGoogle,
			findErr:  oauth2domain.ErrTokenNotFound,
			wantErr:  oauth2domain.ErrTokenNotFound,
		},
		{
			name:     "repo find failure is propagated",
			provider: oauth2domain.ProviderGoogle,
			findErr:  errors.New("dynamodb unavailable"),
			wantErr:  errors.New("dynamodb unavailable"),
		},
		{
			name:              "refresh failure is returned without saving",
			provider:          oauth2domain.ProviderGoogle,
			findToken:         nearExpiryToken,
			refreshErr:        errors.New("provider rejected refresh"),
			wantErr:           errors.New("provider rejected refresh"),
			wantRefreshCalled: true,
		},
		{
			name:              "save failure after refresh is returned",
			provider:          oauth2domain.ProviderGoogle,
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
			client := &mockClient{refreshResult: tt.refreshResult, refreshErr: tt.refreshErr}
			tokens := &mockTokenRepo{findToken: tt.findToken, findErr: tt.findErr, saveErr: tt.saveErr}
			clients := map[oauth2domain.ProviderID]oauth2domain.Client{oauth2domain.ProviderGoogle: client}

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
