package nodefx

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func ExecutionStore(cfg *types.JobStoreConfig) (store.ExecutionStore, error) {
	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	switch cfg.Type {
	case types.BoltDB:
		return boltdb.NewStore(context.TODO(), cfg.Path)
	default:
	}

	panic(fmt.Errorf("unknown JobStore type: %s", cfg.Type))
}
