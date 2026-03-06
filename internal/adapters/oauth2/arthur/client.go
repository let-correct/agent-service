package arthur

import (
	"context"
	"fmt"
	"net/http"

	oauth2Domain "github.com/troysnowden/agent-service/internal/domain/oauth2"

	"golang.org/x/oauth2"
)

type Client struct {
	clientID     string
	clientSecret string
	redirectURL  string
	authURL      string
	tokenURL     string
	httpClient   *http.Client
}

func NewClient(clientID string, clientSecret string, redirectURL string, authURL string, tokenURL string) *Client {
	return &Client{
		clientID: clientID, clientSecret: clientSecret, redirectURL: redirectURL, authURL: authURL, tokenURL: tokenURL,
	}
}

func (c *Client) config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		RedirectURL:  c.redirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  c.authURL,
			TokenURL: c.tokenURL,
		},
	}
}

func (c *Client) AuthorizationURL(_ context.Context, state string) string {
	return c.config().AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (c *Client) ExchangeCode(ctx context.Context, code, email string) (*oauth2Domain.Token, error) {
	if c.httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	}

	aToken, err := c.config().Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}

	return oauth2Domain.NewToken(email, oauth2Domain.ProviderArthur, aToken.AccessToken, aToken.RefreshToken, aToken.Expiry), nil
}

func (c *Client) RefreshToken(ctx context.Context, token *oauth2Domain.Token) (*oauth2Domain.Token, error) {
	if c.httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	}

	// Seed with only the refresh token; zero expiry forces x/oauth2 to refresh immediately.
	expired := &oauth2.Token{RefreshToken: token.RefreshToken()}
	newToken, err := c.config().TokenSource(ctx, expired).Token()
	if err != nil {
		return nil, fmt.Errorf("refreshing token: %w", err)
	}

	return oauth2Domain.NewToken(token.Email(), oauth2Domain.ProviderArthur, newToken.AccessToken, newToken.RefreshToken, newToken.Expiry), nil
}
