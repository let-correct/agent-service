package application

import (
	"context"

	"github.com/troysnowden/let-correct-viewing/internal/domain/oauth2"
)

type ExchangeCodeCommand struct {
	email    string
	provider oauth2.ProviderID
	code     string
}

type ExchangeCode struct {
	clients    map[oauth2.ProviderID]oauth2.Client
	repository oauth2.Repository
}

func (h *ExchangeCode) Handle(ctx context.Context, cmd ExchangeCodeCommand) error {
	client, ok := h.clients[cmd.provider]
	if !ok {
		//return ErrUnsupportedProvider
		return nil
	}

	token, err := client.ExchangeCode(ctx, cmd.code)
	if err != nil {
		return err
	}

	return h.repository.Save(ctx, token)
}
