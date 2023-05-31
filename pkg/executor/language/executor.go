package language

/*
The language executor wraps either the python_wasm executor or the generic
docker executor, depending on whether determinism is required.
*/

import (
	"context"
	"fmt"
	"io"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type Executor struct {
	Jobs               map[string]*model.Job
	executors          executor.ExecutorProvider
	delegatedExecutors generic.SyncMap[string, executor.Executor]
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

func (e *Executor) GetSemanticBidStrategy(ctx context.Context) (bidstrategy.SemanticBidStrategy, error) {
	return semantic.NewChainedSemanticBidStrategy(), nil
}

func (e *Executor) GetResourceBidStrategy(ctx context.Context) (bidstrategy.ResourceBidStrategy, error) {
	return resource.NewChainedResourceBidStrategy(), nil
}

func (e *Executor) Run(
	ctx context.Context,
	executionID string,
	job model.Job,
	jobResultsDir string,
) (*model.RunCommandResult, error) {
	executor, err := e.getDelegateExecutor(ctx, job)
	if err != nil {
		return nil, err
	}
	e.delegatedExecutors.Put(executionID, executor)
	return executor.Run(ctx, executionID, job, jobResultsDir)
}

func (e *Executor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	executor, exists := e.delegatedExecutors.Get(executionID)
	if !exists {
		return nil, fmt.Errorf("execution %v not found", executionID)
	}
	return executor.GetOutputStream(ctx, executionID, withHistory, follow)
}

func (e *Executor) getDelegateExecutor(ctx context.Context, job model.Job) (executor.Executor, error) {
	requiredLang := LanguageSpec{
		Language: job.Spec.Language.Language,
		Version:  job.Spec.Language.LanguageVersion,
	}

	engineKey, exists := supportedVersions[requiredLang]
	if !exists {
		err := fmt.Errorf("%v is not supported", requiredLang)
		return nil, err
	}

	if job.Spec.Language.Deterministic {
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
