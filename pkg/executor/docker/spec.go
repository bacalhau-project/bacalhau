package docker

import (
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

const (
	EngineType                    = "docker"
	EngineKeyImage                = "Image"
	EngineKeyEntrypoint           = "Entrypoint"
	EngineKeyEnvironmentVariables = "EnvironmentVariables"
	EngineKeyWorkingDirectory     = "WorkingDirectory"
)

func NewEngineSpec(image string, entrypoint []string, environmentVariables []string, workingDirectory string) model.EngineSpec {
	return model.EngineSpec{
		Type: EngineType,
		Params: map[string]interface{}{
			EngineKeyImage:                image,
			EngineKeyEntrypoint:           entrypoint,
			EngineKeyEnvironmentVariables: environmentVariables,
			EngineKeyWorkingDirectory:     workingDirectory,
		},
	}
}

// for VM style executors
type Engine struct {
	// this should be pullable by docker
	Image string `json:"Image,omitempty"`
	// optionally override the default entrypoint
	Entrypoint []string `json:"Entrypoint,omitempty"`
	// a map of env to run the container with
	EnvironmentVariables []string `json:"EnvironmentVariables,omitempty"`
	// working directory inside the container
	WorkingDirectory string `json:"WorkingDirectory,omitempty"`
}

func (e Engine) AsEngineSpec() model.EngineSpec {
	return model.EngineSpec{
		Type: EngineType,
		Params: map[string]interface{}{
			EngineKeyImage:                e.Image,
			EngineKeyEntrypoint:           e.Entrypoint,
			EngineKeyEnvironmentVariables: e.EnvironmentVariables,
			EngineKeyWorkingDirectory:     e.WorkingDirectory,
		},
	}
}

func AsEngine(e model.EngineSpec) (Engine, error) {
	if e.Type != EngineType {
		return Engine{}, fmt.Errorf("expected type %s got %s", EngineType, e.Type)
	}
	if e.Params == nil {
		return Engine{}, fmt.Errorf("engine params uninitialized")
	}
	var out Engine
	if err := mapstructure.Decode(e.Params, &out); err != nil {
		return Engine{}, err
	}
	return out, nil
}
