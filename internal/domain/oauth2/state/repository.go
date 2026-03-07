package state

import (
	"context"
	"errors"
)

var ErrStateNotFound = errors.New("state not found")

type Repository interface {
	Save(ctx context.Context, state string) error
	Consume(ctx context.Context, state string) error // validates existence then deletes
}
