package nodefx

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
)

func JobStore(lc fx.Lifecycle, cfg *NodeConfig) (jobstore.Store, error) {
	if err := cfg.RequesterConfig.Store.Validate(); err != nil {
		return nil, err
	}

	var store jobstore.Store
	var err error
	switch cfg.RequesterConfig.Store.Type {
	case types.BoltDB:
		log.Debug().Str("Path", cfg.RequesterConfig.Store.Path).Msg("creating boltdb backed jobstore")
		store, err = boltjobstore.NewBoltJobStore(cfg.RequesterConfig.Store.Path)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", cfg.RequesterConfig.Store.Type)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return store.Close(ctx)
		},
	})
	return store, nil
}
