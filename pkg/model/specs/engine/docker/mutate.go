package docker

import (
	"github.com/bacalhau-project/bacalhau/pkg/model/specs/engine"
)

type MutateOpt func(spec *EngineSpec)

func WithImage(image string) MutateOpt {
	return func(spec *EngineSpec) {
		spec.Image = image
	}
}

func WithEntrypoint(entrypoint ...string) MutateOpt {
	return func(spec *EngineSpec) {
		spec.Entrypoint = entrypoint
	}
}

func WithWorkingDirectory(dir string) MutateOpt {
	return func(spec *EngineSpec) {
		spec.WorkingDirectory = dir
	}
}

func WithEnvironmentVariables(vars ...string) MutateOpt {
	return func(spec *EngineSpec) {
		spec.EnvironmentVariables = vars
	}
}

func AppendEntrypoint(entrypoint ...string) MutateOpt {
	return func(spec *EngineSpec) {
		spec.Entrypoint = append(spec.Entrypoint, entrypoint...)
	}
}

func AppendEnvironmentVariables(envvar ...string) MutateOpt {
	return func(spec *EngineSpec) {
		spec.EnvironmentVariables = append(spec.EnvironmentVariables, envvar...)
	}
}

func Mutate(e engine.Spec, mutations ...MutateOpt) (engine.Spec, error) {
	dockerSpec, err := Decode(e)
	if err != nil {
		return engine.Spec{}, err
	}

	for _, mutate := range mutations {
		mutate(dockerSpec)
	}
	return dockerSpec.AsSpec()
}
