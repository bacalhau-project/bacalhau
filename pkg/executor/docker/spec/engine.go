package spec

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// TODO these are duplicated across the docker executor package and here to avoid dep hell, need a better solution.
const (
	DockerEngineType          = 2
	DockerEngineImageKey      = "Image"
	DockerEngineEntrypointKey = "Entrypoint"
	DockerEngineWorkDirKey    = "WorkingDirectory"
	DockerEngineEnvVarKey     = "EnvironmentVariables"
)

// JobSpecDocker is for VM style executors.
type JobSpecDocker struct {
	// Image is the docker image to run. This must be pull-able by docker.
	Image string `json:"Image,omitempty"`

	// Entrypoint is an optional override for the default container entrypoint.
	Entrypoint []string `json:"Entrypoint,omitempty"`

	// EnvironmentVariables is a map of env to run the container with.
	EnvironmentVariables []string `json:"EnvironmentVariables,omitempty"`

	// WorkingDirectory is the working directory inside the container.
	WorkingDirectory string `json:"WorkingDirectory,omitempty"`
}

func AsJobSpecDocker(e model.EngineSpec) (*JobSpecDocker, error) {
	if e.Type != DockerEngineType {
		return nil, fmt.Errorf("EngineSpec is Type %d, expected %d", e.Type, DockerEngineType)
	}

	if e.Spec == nil {
		return nil, fmt.Errorf("EngineSpec is uninitalized")
	}

	job := &JobSpecDocker{}
	if value, ok := e.Spec[DockerEngineImageKey].(string); ok {
		job.Image = value
	}

	if value, ok := e.Spec[DockerEngineEntrypointKey].([]string); ok {
		for _, v := range value {
			job.Entrypoint = append(job.Entrypoint, v)
		}
	}

	if value, ok := e.Spec[DockerEngineEnvVarKey].([]interface{}); ok {
		for _, v := range value {
			if str, ok := v.(string); ok {
				job.EnvironmentVariables = append(job.EnvironmentVariables, str)
			} else {
				return nil, fmt.Errorf("unable to convert %v to string", v)
			}
		}
	}

	if value, ok := e.Spec[DockerEngineWorkDirKey].(string); ok {
		job.WorkingDirectory = value
	}

	return job, nil
}
