package jobtransform

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// NewEngineMigrator maintains backward compatibility for jobs that were defined with Engine/Wasm/Docker and EngineSpec.
func NewEngineMigrator() Transformer {
	return func(ctx context.Context, job *model.Job) (bool, error) {
		// if we received an "old" job that doesn't use the new model.EngineSpec field
		// migrate it to populate the model.EngineSpec fields.
		//nolint:staticcheck
		if model.IsValidEngine(job.Spec.Engine) {
			return migrateForward(job)
		}
		// else we received a "new" job that uses the model.EngineSpec field, populate deprecated
		// fields.
		return migrateBack(job)
	}
}

// migrateForward populates the model.EngineSpec field iff the engine is a known type.
//
//nolint:staticcheck
func migrateForward(job *model.Job) (bool, error) {
	switch job.Spec.Engine {
	case model.EngineNoop:
		job.Spec.EngineSpec = model.NewEngineBuilder().
			WithType(model.EngineNoop.String()).
			Build()
		return true, nil
	case model.EngineDocker:
		job.Spec.EngineSpec = model.NewDockerEngineBuilder(job.Spec.Docker.Image).
			WithEntrypoint(job.Spec.Docker.Entrypoint...).
			WithParameters(job.Spec.Docker.Parameters...).
			WithWorkingDirectory(job.Spec.Docker.WorkingDirectory).
			WithEnvironmentVariables(job.Spec.Docker.EnvironmentVariables...).
			Build()
		return true, nil
	case model.EngineWasm:
		job.Spec.EngineSpec = model.NewWasmEngineBuilder(job.Spec.Wasm.EntryModule).
			WithEntrypoint(job.Spec.Wasm.EntryPoint).
			WithParameters(job.Spec.Wasm.Parameters...).
			WithImportModules(job.Spec.Wasm.ImportModules...).
			WithEnvironmentVariables(job.Spec.Wasm.EnvironmentVariables).
			Build()
		return true, nil
	default:
		return false, fmt.Errorf("unhandled valied engine: %s", job.Spec.Engine)
	}
}

// migrateBack populates deprecated model.Spec engine fields iff the engine is a known type, else
// no change is made.
//
//nolint:staticcheck
func migrateBack(job *model.Job) (bool, error) {
	// check if it's a know engine type and populate the deprecated fields.
	switch job.Spec.EngineSpec.Engine() {
	case model.EngineNoop:
		job.Spec.Engine = model.EngineNoop
		return true, nil
	case model.EngineDocker:
		dockerSpec, err := model.DecodeEngineSpec[model.DockerEngineSpec](job.Spec.EngineSpec)
		if err != nil {
			return false, fmt.Errorf("decoding docker engine spec: %w", err)
		}
		job.Spec.Engine = model.EngineDocker
		job.Spec.Docker = model.JobSpecDocker{
			Image:                dockerSpec.Image,
			Entrypoint:           dockerSpec.Entrypoint,
			Parameters:           dockerSpec.Parameters,
			EnvironmentVariables: dockerSpec.EnvironmentVariables,
			WorkingDirectory:     dockerSpec.WorkingDirectory,
		}
		return true, nil
	case model.EngineWasm:
		wasmSpec, err := model.DecodeEngineSpec[model.WasmEngineSpec](job.Spec.EngineSpec)
		if err != nil {
			return false, fmt.Errorf("decoding wasm engine spec: %w", err)
		}
		job.Spec.Engine = model.EngineWasm
		job.Spec.Wasm = model.JobSpecWasm{
			EntryModule:          wasmSpec.EntryModule,
			EntryPoint:           wasmSpec.Entrypoint,
			Parameters:           wasmSpec.Parameters,
			EnvironmentVariables: wasmSpec.EnvironmentVariables,
			ImportModules:        wasmSpec.ImportModules,
		}
		return true, nil
	default:
		// received a non-standard engine type, we cannot migrate this, exciting!
		return false, nil
	}
}
