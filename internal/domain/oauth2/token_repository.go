package oauth2

import (
	"context"
	"errors"
)

var ErrTokenNotFound = errors.New("token not found")

type TokenRepository interface {
	Save(ctx context.Context, token *Token) error
	FindByEmailAndProvider(ctx context.Context, email string, provider ProviderID) (*Token, error)
}
