package language

/*
The language executor wraps either the python_wasm executor or the generic
docker executor, depending on whether determinism is required.
*/

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type Executor struct {
	Jobs []*executor.Job

	cm               *system.CleanupManager
	id               string
	storageProviders map[string]storage.StorageProvider
}

func NewExecutor(
	cm *system.CleanupManager,
	id string,
	storageProviders map[string]storage.StorageProvider,
) (*Executor, error) {
	e := &Executor{
		cm:               cm,
		id:               id,
		storageProviders: storageProviders,
	}
	return e, nil
}

func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (e *Executor) HasStorage(ctx context.Context,
	volume storage.StorageSpec) (bool, error) {

	return true, nil
}

func (e *Executor) RunJob(ctx context.Context, job *executor.Job) (
	string, error) {

	if job.Spec.Language.Language != "python" && job.Spec.Language.LanguageVersion != "3.10" {
		return "", fmt.Errorf("only python 3.10 is supported")
	}

	if job.Spec.Language.Deterministic {
		log.Debug().Msgf("running deterministic python 3.10")
		// TODO: Instantiate a python_wasm
	} else {
		log.Debug().Msgf("running arbitrary python 3.10")
		// TODO: Instantiate a docker with python:3.10 image
	}

	e.Jobs = append(e.Jobs, job)
	return "", nil
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
