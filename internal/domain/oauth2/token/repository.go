package token

import (
	"context"
	"errors"

	"github.com/troysnowden/agent-service/internal/domain/oauth2"
)

var ErrTokenNotFound = errors.New("token not found")

type Repository interface {
	Save(ctx context.Context, token *oauth2.Token) error
	FindByEmailAndProvider(ctx context.Context, email string, provider oauth2.ProviderID) (*oauth2.Token, error)
}
