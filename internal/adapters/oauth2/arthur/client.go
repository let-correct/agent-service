package arthur

import (
	oauth2adapter "github.com/troysnowden/agent-service/internal/adapters/oauth2"
	oauth2Domain "github.com/troysnowden/agent-service/internal/domain/oauth2"
	"golang.org/x/oauth2"
)

func NewClient(clientID, clientSecret, redirectURL, authURL, tokenURL string) *oauth2adapter.Client {
	return oauth2adapter.NewClient(clientID, clientSecret, redirectURL, oauth2.Endpoint{AuthURL: authURL, TokenURL: tokenURL}, nil, oauth2Domain.ProviderArthur, nil)
}
