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

func (ds *JobSpecDocker) AsEngineSpec() model.EngineSpec {
	engine := model.EngineSpec{
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

func MutateEngineSpec(e model.EngineSpec, mutate ...func(docker *JobSpecDocker) error) (model.EngineSpec, error) {
	dockerSpec, err := AsJobSpecDocker(e)
	if err != nil {
		return model.EngineSpec{}, err
	}

	for _, m := range mutate {
		if err := m(dockerSpec); err != nil {
			return model.EngineSpec{}, err
		}
	}
	return dockerSpec.AsEngineSpec(), nil
}

func AsJobSpecDocker(e model.EngineSpec) (*JobSpecDocker, error) {
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
	if _, ok := e.Spec[DockerEngineEntrypointKey]; ok {
		if value, ok := e.Spec[DockerEngineEntrypointKey].([]interface{}); ok {
			for _, v := range value {
				if str, ok := v.(string); ok {
					job.Entrypoint = append(job.Entrypoint, str)
				} else {
					return nil, fmt.Errorf("unable to convert %v to string", v)
				}
			}
		} else if value, ok := e.Spec[DockerEngineEntrypointKey].([]string); ok {
			job.Entrypoint = value
		} else {
			return nil, fmt.Errorf("unknow type for docker entrypoint %T", e.Spec[DockerEngineEntrypointKey])
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
