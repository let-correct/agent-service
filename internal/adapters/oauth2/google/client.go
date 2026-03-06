package google

import (
	oauth2adapter "github.com/troysnowden/agent-service/internal/adapters/oauth2"
	oauth2Domain "github.com/troysnowden/agent-service/internal/domain/oauth2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func NewClient(clientID, clientSecret, redirectURL string) *oauth2adapter.Client {
	return oauth2adapter.NewClient(clientID, clientSecret, redirectURL, google.Endpoint, []string{"https://www.googleapis.com/auth/calendar"}, oauth2Domain.ProviderGoogle, []oauth2.AuthCodeOption{oauth2.AccessTypeOffline})
}
