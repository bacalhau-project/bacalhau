package spec

import (
	"encoding/json"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// TODO these are duplicated across the wasm executor package and here to avoid dep hell, need a better solution.
const (
	WasmEngineType             = 3
	WasmEngineEntryModuleKey   = "EntryModule"
	WasmEngineEntryPointKey    = "Entrypoint"
	WasmEngineParametersKey    = "Parameters"
	WasmEngineEnvVarKey        = "EnvironmentVariables"
	WasmEngineImportModulesKey = "ImportModules"
)

// JobSpecWasm describes a raw WASM job.
type JobSpecWasm struct {
	// The module that contains the WASM code to start running.
	EntryModule model.StorageSpec `json:"EntryModule,omitempty"`

	// The name of the function in the EntryModule to call to run the job. For
	// WASI jobs, this will always be `_start`, but jobs can choose to call
	// other WASM functions instead. The EntryPoint must be a zero-parameter
	// zero-result function.
	EntryPoint string `json:"EntryPoint,omitempty"`

	// The arguments supplied to the program (i.e. as ARGV).
	Parameters []string `json:"Parameters,omitempty"`

	// The variables available in the environment of the running program.
	EnvironmentVariables map[string]string `json:"EnvironmentVariables,omitempty"`

	// TODO #880: Other WASM modules whose exports will be available as imports
	// to the EntryModule.
	ImportModules []model.StorageSpec `json:"ImportModules,omitempty"`
}

func (ws *JobSpecWasm) AsEngineSpec() model.EngineSpec {
	engine := model.EngineSpec{
		Type: WasmEngineType,
		Spec: make(map[string]interface{}),
	}

	engine.Spec[WasmEngineEntryModuleKey] = ws.EntryModule
	if ws.EntryPoint != "" {
		engine.Spec[WasmEngineEntryPointKey] = ws.EntryPoint
	}
	if len(ws.Parameters) > 0 {
		engine.Spec[WasmEngineParametersKey] = ws.Parameters
	}
	if len(ws.EnvironmentVariables) > 0 {
		engine.Spec[WasmEngineEnvVarKey] = ws.EnvironmentVariables
	}
	if len(ws.ImportModules) > 0 {
		engine.Spec[WasmEngineImportModulesKey] = ws.ImportModules
	}
	return engine
}

func WithParameters(params ...string) func(wasm *JobSpecWasm) error {
	return func(wasm *JobSpecWasm) error {
		wasm.Parameters = params
		return nil
	}
}

func MutateEngineSpec(e model.EngineSpec, mutate ...func(*JobSpecWasm) error) (model.EngineSpec, error) {
	wasmSpec, err := AsJobSpecWasm(e)
	if err != nil {
		return model.EngineSpec{}, err
	}

	for _, m := range mutate {
		if err := m(wasmSpec); err != nil {
			return model.EngineSpec{}, err
		}
	}
	return wasmSpec.AsEngineSpec(), nil
}

func AsJobSpecWasm(e model.EngineSpec) (*JobSpecWasm, error) {
	if e.Type != WasmEngineType {
		return nil, fmt.Errorf("EngineSpec is Type %s, expected %d", e.Type, WasmEngineType)
	}

	if e.Spec == nil {
		return nil, fmt.Errorf("EngineSpec is uninitalized")
	}

	job := &JobSpecWasm{}

	if entryModule, ok := e.Spec[WasmEngineEntryModuleKey]; ok {
		if value, ok := entryModule.(map[string]interface{}); ok {
			data, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			var storageSpec model.StorageSpec
			if err := json.Unmarshal(data, &storageSpec); err != nil {
				return nil, err
			}
			job.EntryModule = storageSpec
		} else if value, ok := entryModule.(model.StorageSpec); ok {
			job.EntryModule = value
		} else {
			return nil, fmt.Errorf("unknow type %T in %s", value, WasmEngineEntryModuleKey)
		}
	}

	if entryPoint, ok := e.Spec[WasmEngineEntryPointKey]; ok {
		if value, ok := entryPoint.(string); ok {
			job.EntryPoint = value
		} else {
			return nil, fmt.Errorf("unknow type %T in %s", value, WasmEngineEntryPointKey)
		}
	}

	if params, ok := e.Spec[WasmEngineParametersKey]; ok {
		if value, ok := params.([]string); ok {
			for _, v := range value {
				job.Parameters = append(job.Parameters, v)
			}
		} else if value, ok := params.([]interface{}); ok {
			for _, v := range value {
				if str, ok := v.(string); ok {
					job.Parameters = append(job.Parameters, str)
				} else {
					return nil, fmt.Errorf("unable to convert %v to string", v)
				}
			}
		} else {
			return nil, fmt.Errorf("unknow type %T in %s", value, WasmEngineParametersKey)
		}
	}

	if envvar, ok := e.Spec[WasmEngineEnvVarKey]; ok {
		if value, ok := envvar.(map[string]string); ok {
			job.EnvironmentVariables = make(map[string]string)
			for k, v := range value {
				job.EnvironmentVariables[k] = v
			}
		} else {
			return nil, fmt.Errorf("unknow type %T in %s", value, WasmEngineEnvVarKey)
		}
	}

	if importModules, ok := e.Spec[WasmEngineImportModulesKey]; ok {
		// TODO this assertion will probably break whenever this is used, bytes is stating to look more appearing
		if value, ok := importModules.([]model.StorageSpec); ok {
			for _, v := range value {
				job.ImportModules = append(job.ImportModules, v)
			}
		} else {
			return nil, fmt.Errorf("unknow type %T in %s", value, WasmEngineImportModulesKey)
		}
	}

	return job, nil
}
