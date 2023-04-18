package model

import (
	"fmt"
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

func (ds *JobSpecDocker) AsEngineSpec() EngineSpec {
	engine := EngineSpec{
		Type: DockerEngineType,
		Spec: make(map[string]interface{}),
	}

	if ds.Image != "" {
		engine.Spec[DockerEngineImageKey] = ds.Image
	}
	if len(ds.Entrypoint) > 0 {
		engine.Spec[DockerEngineEntrypointKey] = ds.Entrypoint
	}
	if len(ds.EnvironmentVariables) > 0 {
		engine.Spec[DockerEngineEnvVarKey] = ds.EnvironmentVariables
	}
	if ds.WorkingDirectory != "" {
		engine.Spec[DockerEngineWorkDirKey] = ds.WorkingDirectory
	}
	return engine
}

func WithImage(image string) func(*JobSpecDocker) error {
	return func(docker *JobSpecDocker) error {
		docker.Image = image
		return nil
	}
}

func WithEntrypoint(entrypoint ...string) func(*JobSpecDocker) error {
	return func(docker *JobSpecDocker) error {
		docker.Entrypoint = entrypoint
		return nil
	}
}

func MutateDockerEngineSpec(e EngineSpec, mutate ...func(docker *JobSpecDocker) error) (EngineSpec, error) {
	dockerSpec, err := AsJobSpecDocker(e)
	if err != nil {
		return EngineSpec{}, err
	}

	for _, m := range mutate {
		if err := m(dockerSpec); err != nil {
			return EngineSpec{}, err
		}
	}
	return dockerSpec.AsEngineSpec(), nil
}

func AsJobSpecDocker(e EngineSpec) (*JobSpecDocker, error) {
	if e.Type != DockerEngineType {
		return nil, fmt.Errorf("EngineSpec is Type %s, expected %d", e.Type, DockerEngineType)
	}

	if e.Spec == nil {
		return nil, fmt.Errorf("EngineSpec is uninitalized")
	}

	job := &JobSpecDocker{}
	if value, ok := e.Spec[DockerEngineImageKey].(string); ok {
		job.Image = value
	}

	// TODO I think this may be incorrect if there is only a single entry in the value of map.
	if value, ok := e.Spec[DockerEngineEntrypointKey].([]interface{}); ok {
		for _, v := range value {
			if str, ok := v.(string); ok {
				job.Entrypoint = append(job.Entrypoint, str)
			} else {
				return nil, fmt.Errorf("unable to convert %v (%T) to string", v, v)
			}
		}
	} else {
		if value, ok := e.Spec[DockerEngineEntrypointKey].([]string); ok {
			job.Entrypoint = value
		}
	}

	if value, ok := e.Spec[DockerEngineEnvVarKey].([]interface{}); ok {
		for _, v := range value {
			if str, ok := v.(string); ok {
				job.EnvironmentVariables = append(job.EnvironmentVariables, str)
			} else {
				return nil, fmt.Errorf("unable to convert %v (%T) to string", v, v)
			}
		}
	} else if value, ok := e.Spec[DockerEngineEnvVarKey].([]string); ok {
		job.EnvironmentVariables = value

	} else if value, ok := e.Spec[DockerEngineEnvVarKey].(map[string]string); ok {
		for key, val := range value {
			job.EnvironmentVariables = append(job.EnvironmentVariables, key, val)
		}
	} else {
		return nil, fmt.Errorf("DockerEngineEnvVarKey has an unsupported value type %T", value)
	}

	if value, ok := e.Spec[DockerEngineWorkDirKey].(string); ok {
		job.WorkingDirectory = value
	}

	return job, nil
}
