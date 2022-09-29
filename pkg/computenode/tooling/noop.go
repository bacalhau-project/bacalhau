package tooling

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

// return noop for all storage engines
// func NewNoopStorageDrivers(
// 	cm *system.CleanupManager,
// 	config noop_storage.StorageConfig,
// ) (map[model.Engine]executor.Executor, error) {
// 	noopExecutor, err := noop_executor.NewExecutorWithConfig(config)

// 	if err != nil {
// 		return nil, err
// 	}

// 	return map[model.Engine]executor.Executor{
// 		executor.EngineDocker: noopExecutor,
// 		executor.EngineNoop:   noopExecutor,
// 	}, nil
// }

// return noop executors for all engines
func NewNoopExecutors(
	cm *system.CleanupManager,
	config noop_executor.ExecutorConfig,
) (map[model.Engine]executor.Executor, error) {
	noopExecutor, err := noop_executor.NewExecutorWithConfig(config)

	if err != nil {
		return nil, err
	}

	return map[model.Engine]executor.Executor{
		model.EngineDocker: noopExecutor,
		model.EngineNoop:   noopExecutor,
	}, nil
}

func NewNoopExecutor(
	cm *system.CleanupManager,
	config noop_executor.ExecutorConfig,
) (executor.Executor, error) {
	executors, err := NewNoopExecutors(cm, config)
	if err != nil {
		return nil, err
	}
	return executors[model.EngineNoop], nil
}

func BlankNoopExecutorConfig() noop_executor.ExecutorConfig {
	return noop_executor.ExecutorConfig{}
}

func NewNoopExecutorConfig(
	hasStorage bool,
	volumeSize uint64,
	handler noop_executor.ExecutorHandlerJobHandler,
) noop_executor.ExecutorConfig {
	return noop_executor.ExecutorConfig{
		ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
			HasStorageLocally: func(ctx context.Context, volume model.StorageSpec) (bool, error) {
				return hasStorage, nil
			},
			GetVolumeSize: func(ctx context.Context, volume model.StorageSpec) (uint64, error) {
				return volumeSize, nil
			},
			JobHandler: handler,
		},
	}
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
