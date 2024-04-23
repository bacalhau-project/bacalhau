package compute

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

type ExecutionStoreParams struct {
	fx.In

	Config types.JobStoreConfig `name:"execution_store_config"`
}

func ExecutionStore(lc fx.Lifecycle, p ExecutionStoreParams) (store.ExecutionStore, error) {
	if err := p.Config.Validate(); err != nil {
		return nil, err
	}

	var store store.ExecutionStore
	var err error
	switch p.Config.Type {
	case types.BoltDB:
		// TODO(forrest) [refactor]: see TODO in boltdb.NewStore to remove context
		// then "do the thing" to start the store in the lifecycle below
		store, err = boltdb.NewStore(context.TODO(), p.Config.Path)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", p.Config.Type)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return store.Close(ctx)
		},
	})

	return store, nil
}
