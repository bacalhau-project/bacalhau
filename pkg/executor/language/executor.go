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
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type Executor struct {
	Jobs map[string]*model.Job

	executors executor.ExecutorProvider
}

func NewExecutor(
	ctx context.Context,
	cm *system.CleanupManager,
	executors executor.ExecutorProvider,
) (*Executor, error) {
	e := &Executor{
		executors: executors,
	}
	return e, nil
}

func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (e *Executor) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	return true, nil
}

func (e *Executor) GetVolumeSize(ctx context.Context, volumes model.StorageSpec) (uint64, error) {
	return 0, nil
}

func (e *Executor) RunShard(
	ctx context.Context,
	shard model.JobShard,
	jobResultsDir string,
) (*model.RunCommandResult, error) {
	if shard.Job.Spec.Language.Language != "python" && shard.Job.Spec.Language.LanguageVersion != "3.10" {
		err := fmt.Errorf("only python 3.10 is supported")
		return &model.RunCommandResult{ErrorMsg: err.Error()}, err
	}

	if shard.Job.Spec.Language.Deterministic {
		log.Debug().Msgf("running deterministic python 3.10")
		// Instantiate a python_wasm
		// TODO: mutate job as needed?
		pythonWasmExecutor, err := e.executors.GetExecutor(ctx, model.EnginePythonWasm)
		if err != nil {
			return nil, err
		}
		return pythonWasmExecutor.RunShard(ctx, shard, jobResultsDir)
	} else {
		log.Debug().Msgf("running arbitrary python 3.10")
		err := fmt.Errorf("arbitrary python not supported yet")
		// TODO: Instantiate a docker with python:3.10 image
		return &model.RunCommandResult{ErrorMsg: err.Error()}, err
	}
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
