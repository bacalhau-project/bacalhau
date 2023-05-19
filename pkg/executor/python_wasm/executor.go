package pythonwasm

/*
The python_wasm executor wraps the docker executor. The Requester will have
automatically uploaded the execution context (python files, requirements.txt) to
ipfs so that it can be mounted into the wasm runtime container.
*/

import (
	"context"
	"fmt"
	"io"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type Executor struct {
	Jobs map[string]*model.Job

	executors executor.ExecutorProvider
}

func NewExecutor(
	executors executor.ExecutorProvider,
) (*Executor, error) {
	e := &Executor{
		executors: executors,
	}
	return e, nil
}

func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	dockerExecutor, err := e.executors.Get(ctx, model.EngineDocker)
	if err != nil {
		return false, err
	}
	return dockerExecutor.IsInstalled(ctx)
}

func (e *Executor) HasStorageLocally(context.Context, model.StorageSpec) (bool, error) {
	return true, nil
}

func (e *Executor) GetVolumeSize(context.Context, model.StorageSpec) (uint64, error) {
	return 0, nil
}

func (e *Executor) GetSemanticBidStrategy(ctx context.Context) (bidstrategy.SemanticBidStrategy, error) {
	dockerExecutor, err := e.executors.Get(ctx, model.EngineDocker)
	if err != nil {
		return nil, err
	}
	return dockerExecutor.GetSemanticBidStrategy(ctx)
}

func (e *Executor) GetResourceBidStrategy(ctx context.Context) (bidstrategy.ResourceBidStrategy, error) {
	dockerExecutor, err := e.executors.Get(ctx, model.EngineDocker)
	if err != nil {
		return nil, err
	}
	return dockerExecutor.GetResourceBidStrategy(ctx)
}

func (e *Executor) Run(ctx context.Context, executionID string, job model.Job, resultsDir string) (
	*model.RunCommandResult, error) {
	log.Ctx(ctx).Debug().Msgf("in python_wasm executor!")
	// translate language jobspec into a docker run command
	job.Spec.Docker.Image = "ghcr.io/bacalhau-project/pyodide:v0.0.2"
	if job.Spec.Language.Command != "" {
		// pass command through to node wasm wrapper
		job.Spec.Docker.Entrypoint = []string{"node", "n.js", "-c", job.Spec.Language.Command}
	} else if job.Spec.Language.ProgramPath != "" {
		// pass command through to node wasm wrapper
		job.Spec.Docker.Entrypoint = []string{"node", "n.js", fmt.Sprintf("/pyodide_inputs/job/%s", job.Spec.Language.ProgramPath)}
	}
	job.Spec.Engine = model.EngineDocker

	// prepend a path on each of the user supplied volumes to prevent an accidental
	// collision with the internal pyodide filesystem
	for idx, v := range job.Spec.Inputs {
		job.Spec.Inputs[idx].Path = fmt.Sprintf("/pyodide_inputs%s", v.Path)
	}

	for idx, v := range job.Spec.Outputs {
		job.Spec.Outputs[idx].Path = fmt.Sprintf("/pyodide_outputs%s", v.Path)
	}

	// TODO: pass in command, and have n.js interpret it and pass it on to pyodide
	dockerExecutor, err := e.executors.Get(ctx, model.EngineDocker)
	if err != nil {
		return nil, err
	}
	return dockerExecutor.Run(ctx, executionID, job, resultsDir)
}

func (e *Executor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	dockerExecutor, err := e.executors.Get(ctx, model.EngineDocker)
	if err != nil {
		return nil, err
	}
	return dockerExecutor.GetOutputStream(ctx, executionID, withHistory, follow)
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
