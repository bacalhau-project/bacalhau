package executor

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// A simple executor repo that selects a executor based on the job's executor type.
type MappedExecutorProvider struct {
	executors               map[model.Engine]Executor
	executorsInstalledCache map[model.Engine]bool
}

func NewTypeExecutorProvider(executors map[model.Engine]Executor) ExecutorProvider {
	return &MappedExecutorProvider{
		executors:               executors,
		executorsInstalledCache: map[model.Engine]bool{},
	}
}

func (p *MappedExecutorProvider) AddExecutor(ctx context.Context, engineType model.Engine, executor Executor) error {
	_, ok := p.executors[engineType]
	if ok {
		return fmt.Errorf("executor already exists for engine type: %s", engineType)
	}
	p.executors[engineType] = executor
	return nil
}

func (p *MappedExecutorProvider) GetExecutor(ctx context.Context, engineType model.Engine) (Executor, error) {
	executor, ok := p.executors[engineType]
	if !ok {
		return nil, fmt.Errorf(
			"no matching executor found on this server: %s", engineType)
	}

	// cache it being installed so we're not hammering it
	// TODO: we should evict the cache in case an installed executor gets uninstalled, or vice versa
	installed, ok := p.executorsInstalledCache[engineType]
	var err error
	if !ok {
		installed, err = executor.IsInstalled(ctx)
		if err != nil {
			return nil, err
		}
		p.executorsInstalledCache[engineType] = installed
	}

	if !installed {
		return nil, fmt.Errorf("executor is not installed: %s", engineType)
	}

	return executor, nil
}
