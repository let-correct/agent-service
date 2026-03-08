package oauth2

import (
	"context"
	"errors"
)

var ErrStateNotFound = errors.New("state not found")

type StateRepository interface {
	Save(ctx context.Context, state string) error
	Consume(ctx context.Context, state string) error // validates existence then deletes
}
