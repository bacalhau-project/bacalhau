package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type NoopExecutor struct {
	// the global context for stopping any running jobs
	Ctx context.Context

	// are we running in bad actor mode? (useful for tests)
	BadActor bool
}

func NewNoopExecutor(
	ctx context.Context,
	badActor bool,
	storageProviders map[string]storage.StorageProvider,
) (*NoopExecutor, error) {
	NoopExecutor := &NoopExecutor{
		Ctx:      ctx,
		BadActor: badActor,
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
	outputs := []types.StorageSpec{}
	return outputs, nil
}
