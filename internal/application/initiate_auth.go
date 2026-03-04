package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/troysnowden/let-correct-viewing/internal/domain/oauth2"
)

var ErrUnsupportedProvider = errors.New("unsupported provider")

type InitiateAuthCommand struct {
	provider oauth2.ProviderID
}

type InitiateAuthResult struct {
	URL string
}

type InitiateAuth struct {
	clients map[oauth2.ProviderID]oauth2.Client
	states  oauth2.StateRepository
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
