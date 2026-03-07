package application

import (
	"context"

	"github.com/troysnowden/agent-service/internal/domain/oauth2"
	oauthstate "github.com/troysnowden/agent-service/internal/domain/oauth2/state"
	oauthtoken "github.com/troysnowden/agent-service/internal/domain/oauth2/token"
)

type ExchangeCodeCommand struct {
	email    string
	provider oauth2.ProviderID
	code     string
	state    string
}

func NewExchangeCodeCommand(email string, provider oauth2.ProviderID, code, state string) ExchangeCodeCommand {
	return ExchangeCodeCommand{email: email, provider: provider, code: code, state: state}
}

type ExchangeCode struct {
	clients map[oauth2.ProviderID]oauth2.Client
	tokens  oauthtoken.Repository
	states  oauthstate.Repository
}

func NewExchangeCode(clients map[oauth2.ProviderID]oauth2.Client, repository oauthtoken.Repository, states oauthstate.Repository) *ExchangeCode {
	return &ExchangeCode{clients: clients, tokens: repository, states: states}
}

func (h *ExchangeCode) Handle(ctx context.Context, cmd ExchangeCodeCommand) error {
	client, ok := h.clients[cmd.provider]
	if !ok {
		return ErrUnsupportedProvider
	}

	if err := h.states.Consume(ctx, cmd.state); err != nil {
		return err
	}

	token, err := client.ExchangeCode(ctx, cmd.code, cmd.email)
	if err != nil {
		return err
	}

	return h.tokens.Save(ctx, token)
}
