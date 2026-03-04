package oauth2

import "context"

type Client interface {
	AuthorizationURL(ctx context.Context, state string) string
	ExchangeCode(ctx context.Context, code, email string) (*Token, error)
	RefreshToken(ctx context.Context, token *Token) (*Token, error)
}
