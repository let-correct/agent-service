package calendar

import "context"

type SyncRepository interface {
	FindAll(ctx context.Context) ([]*Sync, error)
	Save(ctx context.Context, sync *Sync) error
}
