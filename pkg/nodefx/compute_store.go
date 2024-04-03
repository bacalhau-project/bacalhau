package nodefx

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func ExecutionStore(lc fx.Lifecycle, cfg *ComputeConfig) (store.ExecutionStore, error) {
	if err := cfg.Store.Validate(); err != nil {
		return nil, err
	}

	var store store.ExecutionStore
	var err error
	switch cfg.Store.Type {
	case types.BoltDB:
		store, err = boltdb.NewStore(context.TODO(), cfg.Store.Path)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", cfg.Store.Type)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return store.Close(ctx)
		},
	})

	return store, nil
}
