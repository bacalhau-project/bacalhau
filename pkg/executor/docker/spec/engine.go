package spec

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

const (
	DockerEngineImageKey      = "Image"
	DockerEngineEntrypointKey = "Entrypoint"
	DockerEngineWorkDirKey    = "WorkingDirectory"
)

// for VM style executors
type JobSpecDocker struct {
	// this should be pullable by docker
	Image string `json:"Image,omitempty"`
	// optionally override the default entrypoint
	Entrypoint []string `json:"Entrypoint,omitempty"`
	// working directory inside the container
	WorkingDirectory string `json:"WorkingDirectory,omitempty"`
}

func AsDockerSpec(e model.EngineSpec) (*JobSpecDocker, error) {
	if e.Type != model.EngineDocker {
		return nil, fmt.Errorf("EngineSpec is Type %s, expected %s", e.Type, model.EngineDocker)
	}
	if e.Params == nil {
		return nil, fmt.Errorf("EngineSpec is uninitalized")
	}
	dockerSpec := new(JobSpecDocker)

	if value, ok := e.Params[DockerEngineImageKey].(string); ok {
		dockerSpec.Image = value
	}

	if value, ok := e.Params[DockerEngineEntrypointKey].([]interface{}); ok {
		for _, v := range value {
			if str, ok := v.(string); ok {
				dockerSpec.Entrypoint = append(dockerSpec.Entrypoint, str)
			} else {
				return nil, fmt.Errorf("unable to convert %v to string", v)
			}
		}
	}

	if value, ok := e.Params[DockerEngineWorkDirKey].(string); ok {
		dockerSpec.WorkingDirectory = value
	}

	return dockerSpec, nil
}
