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

func (e *Executor) RunShard(ctx context.Context, shard model.JobShard, resultsDir string) (
	*model.RunCommandResult, error) {
	log.Debug().Msgf("in python_wasm executor!")
	// translate language jobspec into a docker run command
	shard.Job.Spec.Docker.Image = "quay.io/bacalhau/pyodide:e4b0eb7c1d81f320f5b43fc838b0f2a5b9003c9a"
	if shard.Job.Spec.Language.Command != "" {
		// pass command through to node wasm wrapper
		shard.Job.Spec.Docker.Entrypoint = []string{"node", "n.js", "-c", shard.Job.Spec.Language.Command}
	} else if shard.Job.Spec.Language.ProgramPath != "" {
		// pass command through to node wasm wrapper
		shard.Job.Spec.Docker.Entrypoint = []string{"node", "n.js", fmt.Sprintf("/pyodide_inputs/job/%s", shard.Job.Spec.Language.ProgramPath)}
	}
	shard.Job.Spec.Engine = model.EngineDocker

	// prepend a path on each of the user supplied volumes to prevent an accidental
	// collision with the internal pyodide filesystem
	for idx, v := range shard.Job.Spec.InputVolumes {
		shard.Job.Spec.InputVolumes[idx].Path = fmt.Sprintf("/pyodide_inputs%s", v.Path)
	}

	for idx, v := range shard.Job.Spec.Contexts {
		shard.Job.Spec.Contexts[idx].Path = fmt.Sprintf("/pyodide_inputs%s", v.Path)
	}

	for idx, v := range shard.Job.Spec.OutputVolumes {
		shard.Job.Spec.OutputVolumes[idx].Path = fmt.Sprintf("/pyodide_outputs%s", v.Path)
	}

	// TODO: pass in command, and have n.js interpret it and pass it on to pyodide
	return e.executors[model.EngineDocker].RunShard(ctx, shard, resultsDir)
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
