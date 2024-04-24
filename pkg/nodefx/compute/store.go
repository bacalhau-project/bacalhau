package compute

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

type ExecutionStoreParams struct {
	fx.In

	Config types.JobStoreConfig `name:"execution_store_config"`
	Repo   *repo.FsRepo
}

func ExecutionStore(lc fx.Lifecycle, p ExecutionStoreParams) (store.ExecutionStore, error) {
	if err := p.Config.Validate(); err != nil {
		return nil, err
	}

	es, err := p.Repo.ExecutionStore(p.Config)
	if err != nil {
		return nil, fmt.Errorf("creating execution store: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return es.Close(ctx)
		},
	})

	return es, nil
}
