package distributed

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore/commands"
)

type DistributedObjectStore struct {
	callbacks *commands.CallbackHooks
}

func New(options ...Option) (*DistributedObjectStore, error) {
	return &DistributedObjectStore{
		callbacks: commands.NewCallbackHooks(),
	}, nil
}

func (d *DistributedObjectStore) CallbackHooks() *commands.CallbackHooks {
	return d.callbacks
}

func (d *DistributedObjectStore) Get(ctx context.Context, prefix string, key string) ([]byte, error) {
	return []byte{}, nil
}

func (d *DistributedObjectStore) Put(ctx context.Context, prefix string, key string, object any) error {
	return nil
}

func (d *DistributedObjectStore) Close(context.Context) {}
