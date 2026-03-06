package oauth2

import (
	"context"
	"fmt"
	"net/http"

	oauth2Domain "github.com/troysnowden/agent-service/internal/domain/oauth2"
	"golang.org/x/oauth2"
)

type Client struct {
	clientID        string
	clientSecret    string
	redirectURL     string
	endpoint        oauth2.Endpoint
	scopes          []string
	provider        oauth2Domain.ProviderID
	authCodeURLOpts []oauth2.AuthCodeOption
	httpClient      *http.Client
}

func NewClient(clientID, clientSecret, redirectURL string, endpoint oauth2.Endpoint, scopes []string, provider oauth2Domain.ProviderID, authCodeURLOpts []oauth2.AuthCodeOption) *Client {
	return &Client{
		clientID:        clientID,
		clientSecret:    clientSecret,
		redirectURL:     redirectURL,
		endpoint:        endpoint,
		scopes:          scopes,
		provider:        provider,
		authCodeURLOpts: authCodeURLOpts,
	}
}

func (c *Client) config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		RedirectURL:  c.redirectURL,
		Scopes:       c.scopes,
		Endpoint:     c.endpoint,
	}
}

func (c *Client) AuthorizationURL(_ context.Context, state string) string {
	return c.config().AuthCodeURL(state, c.authCodeURLOpts...)
}

func (c *Client) ExchangeCode(ctx context.Context, code, email string) (*oauth2Domain.Token, error) {
	if c.httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	}

	t, err := c.config().Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}

	return oauth2Domain.NewToken(email, c.provider, t.AccessToken, t.RefreshToken, t.Expiry), nil
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

	return oauth2Domain.NewToken(token.Email(), c.provider, newToken.AccessToken, newToken.RefreshToken, newToken.Expiry), nil
}
