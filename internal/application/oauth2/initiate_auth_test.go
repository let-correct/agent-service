package oauth2

import (
	"context"
	"errors"
	"testing"

	oauth2domain "github.com/troysnowden/agent-service/internal/domain/oauth2"
)

func TestInitiateAuth_Handle(t *testing.T) {
	tests := []struct {
		name      string
		provider  oauth2domain.ProviderID
		stateErr  error
		wantErr   error
		wantURL   bool
		wantSaved bool
	}{
		{
			name:      "returns authorization URL for known provider",
			provider:  oauth2domain.ProviderGoogle,
			wantURL:   true,
			wantSaved: true,
		},
		{
			name:     "unknown provider returns ErrUnsupportedProvider without touching state repo",
			provider: oauth2domain.ProviderID("unknown"),
			wantErr:  ErrUnsupportedProvider,
		},
		{
			name:     "state repo failure is returned",
			provider: oauth2domain.ProviderGoogle,
			stateErr: errors.New("dynamodb unavailable"),
			wantErr:  errors.New("dynamodb unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			states := &mockStateRepo{saveErr: tt.stateErr}
			client := &mockClient{authURL: "https://auth.example.com/authorize?state=x"}
			clients := map[oauth2domain.ProviderID]oauth2domain.Client{
				oauth2domain.ProviderGoogle: client,
			}

			h := NewInitiateAuth(clients, states)
			result, err := h.Handle(context.Background(), NewInitiateAuthCommand(tt.provider))

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err)
				}
				if states.saved != "" {
					t.Errorf("state repo should not have been called, but saved %q", states.saved)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantURL && result.URL == "" {
				t.Error("expected non-empty URL, got empty")
			}
			if tt.wantSaved && states.saved == "" {
				t.Error("expected state to be saved, but it was not")
			}
			if client.receivedState != states.saved {
				t.Errorf("state saved %q differs from state passed to client %q", states.saved, client.receivedState)
			}
		})
	}
}
