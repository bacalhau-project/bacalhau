package noop

import (
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type NoopExecutor struct {
	Jobs []*types.Job
}

func NewNoopExecutor() (*NoopExecutor, error) {
	NoopExecutor := &NoopExecutor{
		Jobs: []*types.Job{},
	}
	return NoopExecutor, nil
}

func (noop *NoopExecutor) IsInstalled() (bool, error) {
	return true, nil
}

func (noop *NoopExecutor) HasStorage(volume types.StorageSpec) (bool, error) {
	return true, nil
}

func (noop *NoopExecutor) RunJob(job *types.Job) ([]types.StorageSpec, error) {
	noop.Jobs = append(noop.Jobs, job)
	return []types.StorageSpec{}, nil
}
