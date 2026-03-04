package oauth2

import "context"

type StateRepository interface {
	Save(ctx context.Context, state string) error
	Consume(ctx context.Context, state string) error // validates existence then deletes
}
