package model

import (
	"encoding/json"
	"fmt"
)

const (
	EngineTypeDocker                    = "docker"
	EngineKeyImageDocker                = "image"
	EngineKeyEntrypointDocker           = "entrypoint"
	EngineKeyEnvironmentVariablesDocker = "environmentVariables"
	EngineKeyWorkingDirectoryDocker     = "workingDirectory"
)

// NewDockerEngineSpec returns an EngineSpec of type EngineTypeDocker with the provided arguments as EngineSpec.Params.
func NewDockerEngineSpec(
	image string,
	entrypoint []string,
	environmentVariables []string,
	workingDirectory string,
) EngineSpec {
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

// DockerEngineSpec contains necessary parameters to execute a docker job.
type DockerEngineSpec struct {
	// Image this should be pullable by docker
	Image string `json:"Image,omitempty"`
	// Entrypoint optionally override the default entrypoint
	Entrypoint []string `json:"Entrypoint,omitempty"`
	// EnvironmentVariables is a slice of env to run the container with
	EnvironmentVariables []string `json:"EnvironmentVariables,omitempty"`
	// WorkingDirectory inside the container
	WorkingDirectory string `json:"WorkingDirectory,omitempty"`
}

// AsEngineSpec returns a DockerEngineSpec as an EngineSpec.
func (e DockerEngineSpec) AsEngineSpec() EngineSpec {
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

// DockerEngineSpecFromEngineSpec decodes a DockerEngineSpec from an EngineSpec.
// This method will return an error if:
// - The EngineSpec argument is not of type EngineTypeDocker.
// - The EngineSpec.Params are nil.
// - The EngineSpec.Params cannot be marshaled to json bytes.
// - The EngineSpec.Params cannot be unmarshalled to a DockerEngineSpec.
func DockerEngineSpecFromEngineSpec(e EngineSpec) (DockerEngineSpec, error) {
	if e.Type != EngineTypeDocker {
		return DockerEngineSpec{}, fmt.Errorf("expected type %s got %s", EngineTypeDocker, e.Type)
	}
	if e.Params == nil {
		return DockerEngineSpec{}, fmt.Errorf("engine params uninitialized")
	}
	// NB(forrest): we rely on go's json marshaller to handle the conversion of e.Params map[string]interface{} to the
	// typed structure DockerEngineSpec.
	eb, err := json.Marshal(e.Params)
	if err != nil {
		return DockerEngineSpec{}, nil
	}
	var out DockerEngineSpec
	if err := json.Unmarshal(eb, &out); err != nil {
		return DockerEngineSpec{}, err
	}
	return out, nil
}
