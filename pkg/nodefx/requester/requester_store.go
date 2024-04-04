package requester

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

func JobStore(lc fx.Lifecycle, cfg node.RequesterConfig) (jobstore.Store, error) {
	if err := cfg.JobStoreConfig.Validate(); err != nil {
		return nil, err
	}

	var store jobstore.Store
	var err error
	switch cfg.JobStoreConfig.Type {
	case types.BoltDB:
		log.Debug().Str("Path", cfg.JobStoreConfig.Path).Msg("creating boltdb backed jobstore")
		store, err = boltjobstore.NewBoltJobStore(cfg.JobStoreConfig.Path)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", cfg.JobStoreConfig.Type)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return store.Close(ctx)
		},
	})
	return store, nil
}
