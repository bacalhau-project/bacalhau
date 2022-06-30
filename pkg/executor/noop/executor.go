package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
)

type Executor struct {
	Jobs []*executor.Job
}

func NewExecutor() (*Executor, error) {
	Executor := &Executor{
		Jobs: []*executor.Job{},
	}
	return Executor, nil
}

func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (e *Executor) HasStorage(ctx context.Context, volume storage.StorageSpec) (bool, error) {
	return true, nil
}

func (e *Executor) RunJob(ctx context.Context, job *executor.Job) (string, error) {
	e.Jobs = append(e.Jobs, job)
	return "", nil
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
