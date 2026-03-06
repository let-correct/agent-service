package oauth2

import "context"

type TokenRepository interface {
	Save(ctx context.Context, token *Token) error
	FindByEmailAndProvider(ctx context.Context, email string, provider ProviderID) (*Token, error)
}
