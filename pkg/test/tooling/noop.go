package tooling

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

// return noop executors for all engines
func NewNoopExecutors(
	cm *system.CleanupManager,
	config noop_executor.ExecutorConfig,
) (map[executor.EngineType]executor.Executor, error) {
	noopExecutor, err := noop_executor.NewExecutorWithConfig(config)

	if err != nil {
		return nil, err
	}

	return map[executor.EngineType]executor.Executor{
		executor.EngineDocker: noopExecutor,
		executor.EngineNoop:   noopExecutor,
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
	return executors[executor.EngineNoop], nil
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
			HasStorageLocally: func(ctx context.Context, volume storage.StorageSpec) (bool, error) {
				return hasStorage, nil
			},
			GetVolumeSize: func(ctx context.Context, volume storage.StorageSpec) (uint64, error) {
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
			HasStorageLocally: func(ctx context.Context, volume storage.StorageSpec) (bool, error) {
				return hasStorage, nil
			},
		},
	}
}
