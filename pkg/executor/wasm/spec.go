package wasm

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

const (
	EngineType                    = "wasm"
	EngineKeyEntryModule          = "entrymodule"
	EngineKeyEntrypoint           = "entrypoint"
	EngineKeyParameters           = "parameters"
	EngineKeyEnvironmentVariables = "environmentvariables"
	EngineKeyImportModules        = "importmodules"
)

func NewEngineSpec(
	entryModule model.StorageSpec,
	entrypoint string,
	parameters []string,
	environmentVariables map[string]string,
	importModules []model.StorageSpec) model.EngineSpec {
	return model.EngineSpec{
		Type: EngineType,
		Params: map[string]interface{}{
			EngineKeyEntryModule:          entryModule,
			EngineKeyEntrypoint:           entrypoint,
			EngineKeyParameters:           parameters,
			EngineKeyEnvironmentVariables: environmentVariables,
			EngineKeyImportModules:        importModules,
		},
	}
}

// Describes a raw WASM job
type Engine struct {
	// The module that contains the WASM code to start running.
	EntryModule model.StorageSpec `json:"EntryModule,omitempty"`

	// The name of the function in the EntryModule to call to run the job. For
	// WASI jobs, this will always be `_start`, but jobs can choose to call
	// other WASM functions instead. The EntryPoint must be a zero-parameter
	// zero-result function.
	Entrypoint string `json:"EntryPoint,omitempty"`

	// The arguments supplied to the program (i.e. as ARGV).
	Parameters []string `json:"Parameters,omitempty"`

	// The variables available in the environment of the running program.
	EnvironmentVariables map[string]string `json:"EnvironmentVariables,omitempty"`

	// TODO #880: Other WASM modules whose exports will be available as imports
	// to the EntryModule.
	ImportModules []model.StorageSpec `json:"ImportModules,omitempty"`
}

func (e Engine) AsEngineSpec() model.EngineSpec {
	return model.EngineSpec{
		Type: EngineType,
		Params: map[string]interface{}{
			EngineKeyEntryModule:          e.EntryModule,
			EngineKeyEntrypoint:           e.Entrypoint,
			EngineKeyParameters:           e.Parameters,
			EngineKeyEnvironmentVariables: e.EnvironmentVariables,
			EngineKeyImportModules:        e.ImportModules,
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
	return Engine{
		EntryModule:          e.Params[EngineKeyEntryModule].(model.StorageSpec),
		Entrypoint:           e.Params[EngineKeyEntrypoint].(string),
		Parameters:           e.Params[EngineKeyParameters].([]string),
		EnvironmentVariables: e.Params[EngineKeyEnvironmentVariables].(map[string]string),
		ImportModules:        e.Params[EngineKeyImportModules].([]model.StorageSpec),
	}, nil
}
