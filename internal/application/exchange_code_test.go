package application

import (
	"context"
	"errors"
	"testing"

	"github.com/troysnowden/agent-service/internal/domain/oauth2"
)

type mockTokenRepo struct {
	saved   bool
	saveErr error
}

func (m *mockTokenRepo) Save(_ context.Context, _ *oauth2.Token) error {
	m.saved = true
	return m.saveErr
}

func (m *mockTokenRepo) FindByEmailAndProvider(_ context.Context, _ string, _ oauth2.ProviderID) (*oauth2.Token, error) {
	return nil, nil
}

func TestExchangeCode_Handle(t *testing.T) {
	tests := []struct {
		name        string
		provider    oauth2.ProviderID
		consumeErr  error
		exchangeErr error
		saveErr     error
		wantErr     error
		// behavioural assertions
		wantConsumed  bool
		wantExchanged bool
		wantSaved     bool
	}{
		{
			name:          "success: state consumed, token exchanged and saved",
			provider:      oauth2.ProviderGoogle,
			wantConsumed:  true,
			wantExchanged: true,
			wantSaved:     true,
		},
		{
			name:     "unknown provider returns ErrUnsupportedProvider without touching state or client",
			provider: oauth2.ProviderID("unknown"),
			wantErr:  ErrUnsupportedProvider,
		},
		{
			name:         "invalid state aborts before exchange",
			provider:     oauth2.ProviderGoogle,
			consumeErr:   oauth2.ErrStateNotFound,
			wantErr:      oauth2.ErrStateNotFound,
			wantConsumed: true,
		},
		{
			name:          "client exchange failure is returned without saving token",
			provider:      oauth2.ProviderGoogle,
			exchangeErr:   errors.New("provider rejected code"),
			wantErr:       errors.New("provider rejected code"),
			wantConsumed:  true,
			wantExchanged: true,
		},
		{
			name:          "token repo failure is returned",
			provider:      oauth2.ProviderGoogle,
			saveErr:       errors.New("dynamodb unavailable"),
			wantErr:       errors.New("dynamodb unavailable"),
			wantConsumed:  true,
			wantExchanged: true,
			wantSaved:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			states := &mockStateRepo{consumeErr: tt.consumeErr}
			tokens := &mockTokenRepo{saveErr: tt.saveErr}
			client := &mockClient{exchangeErr: tt.exchangeErr}
			clients := map[oauth2.ProviderID]oauth2.Client{
				oauth2.ProviderGoogle: client,
			}

			h := NewExchangeCode(clients, tokens, states)
			cmd := NewExchangeCodeCommand("user@example.com", tt.provider, "auth-code", "state-token")
			err := h.Handle(context.Background(), cmd)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantConsumed && !states.consumed {
				t.Error("expected state to be consumed, but it was not")
			}
			if !tt.wantConsumed && states.consumed {
				t.Error("state should not have been consumed")
			}
			if tt.wantExchanged && !client.exchangeCalled {
				t.Error("expected code exchange to be called, but it was not")
			}
			if !tt.wantExchanged && client.exchangeCalled {
				t.Error("code exchange should not have been called")
			}
			if tt.wantSaved && !tokens.saved {
				t.Error("expected token to be saved, but it was not")
			}
			if !tt.wantSaved && tokens.saved {
				t.Error("token should not have been saved")
			}
		})
	}
}
