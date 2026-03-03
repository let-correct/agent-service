package oauth2

import "context"

type Client interface {
	ExchangeCode(ctx context.Context, code string) (*Token, error)
	RefreshToken(ctx context.Context, token *Token) (*Token, error)
}
