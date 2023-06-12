package distributed

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore/index"
)

type DistributedObjectStore struct {
	callbacks *index.CallbackHooks
}

func New(options ...Option) (*DistributedObjectStore, error) {
	return &DistributedObjectStore{
		callbacks: index.NewCallbackHooks(),
	}, nil
}

func (d *DistributedObjectStore) CallbackHooks() *index.CallbackHooks {
	return d.callbacks
}

func (d *DistributedObjectStore) Delete(ctx context.Context, prefix string, key string, object any) error {
	return nil
}

func (d *DistributedObjectStore) GetBatch(ctx context.Context, prefix string, keys []string, objects any) error {
	return nil
}

func (d *DistributedObjectStore) Get(ctx context.Context, prefix string, key string, object any) error {
	return nil
}

func (d *DistributedObjectStore) Put(ctx context.Context, prefix string, key string, object any) error {
	return nil
}

func (d *DistributedObjectStore) Close(context.Context) error {
	return nil
}
