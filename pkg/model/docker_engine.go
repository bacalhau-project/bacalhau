package model

import (
	"fmt"
)

const (
	EngineTypeDocker                    = "docker"
	EngineKeyImageDocker                = "Image"
	EngineKeyEntrypointDocker           = "Entrypoint"
	EngineKeyEnvironmentVariablesDocker = "EnvironmentVariables"
	EngineKeyWorkingDirectoryDocker     = "WorkingDirectory"
)

func NewDockerEngineSpec(image string, entrypoint []string, environmentVariables []string, workingDirectory string) EngineSpec {
	return EngineSpec{
		Type: EngineTypeDocker,
		Params: map[string]interface{}{
			EngineKeyImageDocker:                image,
			EngineKeyEntrypointDocker:           entrypoint,
			EngineKeyEnvironmentVariablesDocker: environmentVariables,
			EngineKeyWorkingDirectoryDocker:     workingDirectory,
		},
	}
}

// for VM style executors
type DockerEngine struct {
	// this should be pullable by docker
	Image string `json:"Image,omitempty"`
	// optionally override the default entrypoint
	Entrypoint []string `json:"Entrypoint,omitempty"`
	// a map of env to run the container with
	EnvironmentVariables []string `json:"EnvironmentVariables,omitempty"`
	// working directory inside the container
	WorkingDirectory string `json:"WorkingDirectory,omitempty"`
}

func (e DockerEngine) AsEngineSpec() EngineSpec {
	return EngineSpec{
		Type: EngineTypeDocker,
		Params: map[string]interface{}{
			EngineKeyImageDocker:                e.Image,
			EngineKeyEntrypointDocker:           e.Entrypoint,
			EngineKeyEnvironmentVariablesDocker: e.EnvironmentVariables,
			EngineKeyWorkingDirectoryDocker:     e.WorkingDirectory,
		},
	}
}

func DockerEngineFromEngineSpec(e EngineSpec) (DockerEngine, error) {
	if e.Type != EngineTypeDocker {
		return DockerEngine{}, fmt.Errorf("expected type %s got %s", EngineTypeDocker, e.Type)
	}
	if e.Params == nil {
		return DockerEngine{}, fmt.Errorf("engine params uninitialized")
	}
	return DockerEngine{
		Image:                e.Params[EngineKeyImageDocker].(string),
		Entrypoint:           e.Params[EngineKeyEntrypointDocker].([]string),
		EnvironmentVariables: e.Params[EngineKeyEnvironmentVariablesDocker].([]string),
		WorkingDirectory:     e.Params[EngineKeyWorkingDirectoryDocker].(string),
	}, nil
}
