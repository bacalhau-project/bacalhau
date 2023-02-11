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

type LanguageSpec struct {
	Language, Version string
}

var supportedVersions = map[LanguageSpec]model.Engine{
	{"python", "3.10"}: model.EnginePythonWasm,
	{"wasm", "2.0"}:    model.EngineWasm,
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
	executor, err := e.getDelegateExecutor(ctx, shard)
	if err != nil {
		return nil, err
	}
	return executor.RunShard(ctx, shard, jobResultsDir)
}

func (e *Executor) getDelegateExecutor(ctx context.Context, shard model.JobShard) (executor.Executor, error) {
	requiredLang := LanguageSpec{
		Language: shard.Job.Spec.Language.Language,
		Version:  shard.Job.Spec.Language.LanguageVersion,
	}

	engineKey, exists := supportedVersions[requiredLang]
	if !exists {
		err := fmt.Errorf("%v is not supported", requiredLang)
		return nil, err
	}

	if shard.Job.Spec.Language.Deterministic {
		log.Ctx(ctx).Debug().Msgf("Running deterministic %v", requiredLang)
		// Instantiate a python_wasm
		// TODO: mutate job as needed?
		executor, err := e.executors.Get(ctx, engineKey)
		if err != nil {
			return nil, err
		}
		return executor, nil
	} else {
		err := fmt.Errorf("non-deterministic %v not supported yet", requiredLang)
		// TODO: Instantiate a docker with python:3.10 image
		return nil, err
	}
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
