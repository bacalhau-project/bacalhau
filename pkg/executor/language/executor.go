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
	Jobs []*model.Job

	executors map[model.EngineType]executor.Executor
}

func NewExecutor(
	ctx context.Context,
	cm *system.CleanupManager,
	executors map[model.EngineType]executor.Executor,
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
) *model.RunExecutorResult {
	if shard.Job.Spec.Language.Language != "python" && shard.Job.Spec.Language.LanguageVersion != "3.10" {
		return &model.RunExecutorResult{RunnerError: fmt.Errorf("only python 3.10 is supported")}
	}

	if shard.Job.Spec.Language.Deterministic {
		log.Debug().Msgf("running deterministic python 3.10")
		// Instantiate a python_wasm
		// TODO: mutate job as needed?
		return e.executors[model.EnginePythonWasm].RunShard(ctx, shard, jobResultsDir)
	} else {
		log.Debug().Msgf("running arbitrary python 3.10")
		// TODO: Instantiate a docker with python:3.10 image
		return &model.RunExecutorResult{RunnerError: fmt.Errorf("arbitrary python not supported yet")}
	}
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
