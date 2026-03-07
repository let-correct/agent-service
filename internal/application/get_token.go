package application

import (
	"context"
	"time"

	"github.com/troysnowden/agent-service/internal/domain/oauth2"
)

type GetTokenCommand struct {
	email    string
	provider oauth2.ProviderID
}

func NewGetTokenCommand(email string, provider oauth2.ProviderID) GetTokenCommand {
	return GetTokenCommand{email: email, provider: provider}
}

type GetTokenResult struct {
	AccessToken string
	ExpiresAt   time.Time
}

type GetToken struct {
	clients map[oauth2.ProviderID]oauth2.Client
	tokens  oauth2.TokenRepository
}

func NewGetToken(clients map[oauth2.ProviderID]oauth2.Client, tokens oauth2.TokenRepository) *GetToken {
	return &GetToken{clients: clients, tokens: tokens}
}

func (h *GetToken) Handle(ctx context.Context, cmd GetTokenCommand) (GetTokenResult, error) {
	client, ok := h.clients[cmd.provider]
	if !ok {
		return GetTokenResult{}, ErrUnsupportedProvider
	}

	token, err := h.tokens.FindByEmailAndProvider(ctx, cmd.email, cmd.provider)
	if err != nil {
		return GetTokenResult{}, err
	}

	if token.NeedsRefresh() {
		token, err = client.RefreshToken(ctx, token)
		if err != nil {
			return GetTokenResult{}, err
		}
		if err := h.tokens.Save(ctx, token); err != nil {
			return GetTokenResult{}, err
		}
	}

	return GetTokenResult{
		AccessToken: token.AccessToken(),
		ExpiresAt:   token.ExpiresAt(),
	}, nil
}
