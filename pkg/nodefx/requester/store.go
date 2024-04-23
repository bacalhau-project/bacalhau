package requester

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
)

type JobStoreParams struct {
	fx.In

	Config types.JobStoreConfig `name:"job_store_config"`
}

func JobStore(lc fx.Lifecycle, p JobStoreParams) (jobstore.Store, error) {
	if err := p.Config.Validate(); err != nil {
		return nil, err
	}

	var store jobstore.Store
	var err error
	switch p.Config.Type {
	case types.BoltDB:
		log.Debug().Str("Path", p.Config.Path).Msg("creating boltdb backed jobstore")
		store, err = boltjobstore.NewBoltJobStore(p.Config.Path)
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
