package pythonwasm

/*
The python_wasm executor wraps the docker executor. The requestor will have
automatically uploaded the execution context (python files, requirements.txt) to
ipfs so that it can be mounted into the wasm runtime container.
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

	executors map[executor.EngineType]executor.Executor
}

func NewExecutor(
	cm *system.CleanupManager,
	executors map[executor.EngineType]executor.Executor,
) (*Executor, error) {
	e := &Executor{
		executors: executors,
	}
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

func (e *Executor) RunJob(ctx context.Context, job *executor.Job) (
	string, error) {
	log.Debug().Msgf("in python_wasm executor!")
	// translate language jobspec into a docker run command
	job.Spec.Docker.Image = "quay.io/bacalhau/pyodide:127391ad9c1a0d69bf16bb5c48bb9ba05328fed1"
	if job.Spec.Language.Command != "" {
		// pass command through to node wasm wrapper
		job.Spec.Docker.Entrypoint = []string{"node", "n.js", "-c", job.Spec.Language.Command}
	} else if job.Spec.Language.ProgramPath != "" {
		// pass command through to node wasm wrapper
		job.Spec.Docker.Entrypoint = []string{"node", "n.js", fmt.Sprintf("/job/%s", job.Spec.Language.ProgramPath)}
	}
	job.Spec.Engine = executor.EngineDocker
	// TODO: pass in command, and have n.js interpret it and pass it on to pyodide
	return e.executors[executor.EngineDocker].RunJob(ctx, job)
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
