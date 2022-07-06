package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
)

type ExecutorConfigExternalHooks struct {
	JobHandler func(ctx context.Context, job *executor.Job) (string, error)
}

type ExecutorConfig struct {
	ExternalHooks *ExecutorConfigExternalHooks
}

type Executor struct {
	Jobs   []*executor.Job
	Config ExecutorConfig
}

func NewExecutor() (*Executor, error) {
	Executor := &Executor{
		Jobs: []*executor.Job{},
	}
	return Executor, nil
}

func NewExecutorWithConfig(config ExecutorConfig) (*Executor, error) {
	e, err := NewExecutor()
	if err != nil {
		return nil, err
	}
	e.Config = config
	return e, nil
}

func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (e *Executor) HasStorageLocally(ctx context.Context, volume storage.StorageSpec) (bool, error) {
	return true, nil
}

func (e *Executor) GetVolumeSize(ctx context.Context, volumes storage.StorageSpec) (uint64, error) {
	return 0, nil
}

func (e *Executor) RunJob(ctx context.Context, job *executor.Job) (string, error) {
	e.Jobs = append(e.Jobs, job)
	if e.Config.ExternalHooks != nil {
		return e.Config.ExternalHooks.JobHandler(ctx, job)
	}
	return "", nil
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
