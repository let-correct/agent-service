package google

import (
	"context"
	"net/http"

	"github.com/troysnowden/let-correct-viewing/internal/domain/oauth2"
)

type Client struct {
	clientID     string
	clientSecret string
	redirectURL  string
	httpClient   *http.Client
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	// implement
	return &oauth2.Token{}, nil
}

func (c *Client) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	// implement
	return &oauth2.Token{}, nil
}
