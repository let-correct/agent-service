package oauth2

import (
	"context"
	"time"

	oauth2domain "github.com/troysnowden/agent-service/internal/domain/oauth2"
)

type mockClient struct {
	authURL        string
	receivedState  string
	exchangeCalled bool
	exchangeErr    error
	refreshCalled  bool
	refreshResult  *oauth2domain.Token
	refreshErr     error
}

func (m *mockClient) AuthorizationURL(_ context.Context, state string) string {
	m.receivedState = state
	return m.authURL
}

func (m *mockClient) ExchangeCode(_ context.Context, _, email string) (*oauth2domain.Token, error) {
	m.exchangeCalled = true
	if m.exchangeErr != nil {
		return nil, m.exchangeErr
	}
	return oauth2domain.NewToken(email, oauth2domain.ProviderGoogle, "access", "refresh", time.Now().Add(time.Hour)), nil
}

func (m *mockClient) RefreshToken(_ context.Context, token *oauth2domain.Token) (*oauth2domain.Token, error) {
	m.refreshCalled = true
	if m.refreshErr != nil {
		return nil, m.refreshErr
	}
	if m.refreshResult != nil {
		return m.refreshResult, nil
	}
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

type mockTokenRepo struct {
	findToken *oauth2domain.Token
	findErr   error
	saved     bool
	saveErr   error
}

func (m *mockTokenRepo) Save(_ context.Context, _ *oauth2domain.Token) error {
	m.saved = true
	return m.saveErr
}

func (m *mockTokenRepo) FindByEmailAndProvider(_ context.Context, _ string, _ oauth2domain.ProviderID) (*oauth2domain.Token, error) {
	return m.findToken, m.findErr
}
