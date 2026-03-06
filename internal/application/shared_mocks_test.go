package application

import (
	"context"
	"time"

	"github.com/troysnowden/agent-service/internal/domain/oauth2"
)

type mockClient struct {
	authURL        string
	receivedState  string
	exchangeCalled bool
	exchangeErr    error
}

func (m *mockClient) AuthorizationURL(_ context.Context, state string) string {
	m.receivedState = state
	return m.authURL
}

func (m *mockClient) ExchangeCode(_ context.Context, _, email string) (*oauth2.Token, error) {
	m.exchangeCalled = true
	if m.exchangeErr != nil {
		return nil, m.exchangeErr
	}
	return oauth2.NewToken(email, oauth2.ProviderGoogle, "access", "refresh", time.Now().Add(time.Hour)), nil
}

func (m *mockClient) RefreshToken(_ context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	return token, nil
}

type mockStateRepo struct {
	saved      string
	consumed   bool
	saveErr    error
	consumeErr error
}

func (m *mockStateRepo) Save(_ context.Context, state string) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = state
	return nil
}

func (m *mockStateRepo) Consume(_ context.Context, _ string) error {
	m.consumed = true
	return m.consumeErr
}
