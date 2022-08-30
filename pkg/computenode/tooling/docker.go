package tooling

import (
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

// get a docker executor with the given storage drivers
func NewDockerExecutors(
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

func NewDockerExecutor(
	cm *system.CleanupManager,
	config noop_executor.ExecutorConfig,
) (executor.Executor, error) {
	executors, err := NewNoopExecutors(cm, config)
	if err != nil {
		return nil, err
	}
	return executors[executor.EngineNoop], nil
}
