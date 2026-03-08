package oauth2

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	oauth2domain "github.com/troysnowden/agent-service/internal/domain/oauth2"
)

var ErrUnsupportedProvider = errors.New("unsupported provider")

type InitiateAuthCommand struct {
	provider oauth2domain.ProviderID
}

type InitiateAuthResult struct {
	URL string
}

type InitiateAuth struct {
	clients map[oauth2domain.ProviderID]oauth2domain.Client
	states  oauth2domain.StateRepository
}

func NewInitiateAuth(clients map[oauth2domain.ProviderID]oauth2domain.Client, states oauth2domain.StateRepository) *InitiateAuth {
	return &InitiateAuth{clients: clients, states: states}
}

func NewInitiateAuthCommand(provider oauth2domain.ProviderID) InitiateAuthCommand {
	return InitiateAuthCommand{provider: provider}
}

func (h *InitiateAuth) Handle(ctx context.Context, cmd InitiateAuthCommand) (InitiateAuthResult, error) {
	client, ok := h.clients[cmd.provider]
	if !ok {
		return InitiateAuthResult{}, ErrUnsupportedProvider
	}

	state, err := generateState()
	if err != nil {
		return InitiateAuthResult{}, err
	}

	if err := h.states.Save(ctx, state); err != nil {
		return InitiateAuthResult{}, err
	}

	return InitiateAuthResult{URL: client.AuthorizationURL(ctx, state)}, nil
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
