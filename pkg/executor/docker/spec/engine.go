package spec

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
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
	if e.Type != model.EngineDocker {
		return nil, fmt.Errorf("EngineSpec is Type %s, expected %s", e.Type, model.EngineDocker)
	}

	if e.Spec == nil {
		return nil, fmt.Errorf("EngineSpec is uninitalized")
	}

	job := &JobSpecDocker{}
	if value, ok := e.Spec["Image"].(string); ok {
		job.Image = value
	}

	if value, ok := e.Spec["Entrypoint"].([]string); ok {
		for _, v := range value {
			job.Entrypoint = append(job.Entrypoint, v)
		}
	}

	if value, ok := e.Spec["EnvironmentVariables"].([]interface{}); ok {
		for _, v := range value {
			if str, ok := v.(string); ok {
				job.EnvironmentVariables = append(job.EnvironmentVariables, str)
			} else {
				return nil, fmt.Errorf("unable to convert %v to string", v)
			}
		}
	}

	if value, ok := e.Spec["WorkingDirectory"].(string); ok {
		job.WorkingDirectory = value
	}

	return job, nil
}
