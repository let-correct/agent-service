package calendarsync

import "context"

type Repository interface {
	FindAll(ctx context.Context) ([]*Sync, error)
	Save(ctx context.Context, sync *Sync) error
}
