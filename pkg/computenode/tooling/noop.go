package tooling

import (
	"context"

	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

func BlankNoopExecutorConfig() noop_executor.ExecutorConfig {
	return noop_executor.ExecutorConfig{}
}

func HasStorageNoopExecutorConfig(
	hasStorage bool,
) noop_executor.ExecutorConfig {
	return noop_executor.ExecutorConfig{
		ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
			HasStorageLocally: func(ctx context.Context, volume model.StorageSpec) (bool, error) {
				return hasStorage, nil
			},
		},
	}
}
