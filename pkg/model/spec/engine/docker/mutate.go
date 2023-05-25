package docker

import (
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine"
)

type MutateOpt func(spec *DockerEngineSpec)

func WithImage(image string) MutateOpt {
	return func(spec *DockerEngineSpec) {
		spec.Image = image
	}
}

func WithEntrypoint(entrypoint ...string) MutateOpt {
	return func(spec *DockerEngineSpec) {
		spec.Entrypoint = entrypoint
	}
}

func WithWorkingDirectory(dir string) MutateOpt {
	return func(spec *DockerEngineSpec) {
		spec.WorkingDirectory = dir
	}
}

func WithEnvironmentVariables(vars ...string) MutateOpt {
	return func(spec *DockerEngineSpec) {
		spec.EnvironmentVariables = vars
	}
}

func AppendEntrypoint(entrypoint ...string) MutateOpt {
	return func(spec *DockerEngineSpec) {
		spec.Entrypoint = append(spec.Entrypoint, entrypoint...)
	}
}

func AppendEnvironmentVariables(envvar ...string) MutateOpt {
	return func(spec *DockerEngineSpec) {
		spec.EnvironmentVariables = append(spec.EnvironmentVariables, envvar...)
	}
}

func Mutate(e engine.Engine, mutations ...MutateOpt) (engine.Engine, error) {
	dockerSpec, err := Decode(e)
	if err != nil {
		return engine.Engine{}, err
	}

	for _, mutate := range mutations {
		mutate(dockerSpec)
	}
	return dockerSpec.AsSpec()
}
