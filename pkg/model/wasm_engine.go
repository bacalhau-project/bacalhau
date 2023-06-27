package model

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

const (
	EngineTypeWasm                    = "wasm"
	EngineKeyEntryModuleWasm          = "entrymodule"
	EngineKeyEntrypointWasm           = "entrypoint"
	EngineKeyParametersWasm           = "parameters"
	EngineKeyEnvironmentVariablesWasm = "environmentvariables"
	EngineKeyImportModulesWasm        = "importmodules"
)

func NewWasmEngineSpec(
	entryModule StorageSpec,
	entrypoint string,
	parameters []string,
	environmentVariables map[string]string,
	importModules []StorageSpec) EngineSpec {
	return EngineSpec{
		Type: EngineTypeWasm,
		Params: map[string]interface{}{
			EngineKeyEntryModuleWasm:          entryModule,
			EngineKeyEntrypointWasm:           entrypoint,
			EngineKeyParametersWasm:           parameters,
			EngineKeyEnvironmentVariablesWasm: environmentVariables,
			EngineKeyImportModulesWasm:        importModules,
		},
	}
}

// Describes a raw WASM job
type WasmEngine struct {
	// The module that contains the WASM code to start running.
	EntryModule StorageSpec `json:"EntryModule,omitempty"`

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
	ImportModules []StorageSpec `json:"ImportModules,omitempty"`
}

func (e WasmEngine) AsEngineSpec() EngineSpec {
	return EngineSpec{
		Type: EngineTypeWasm,
		Params: map[string]interface{}{
			EngineKeyEntryModuleWasm:          e.EntryModule,
			EngineKeyEntrypointWasm:           e.Entrypoint,
			EngineKeyParametersWasm:           e.Parameters,
			EngineKeyEnvironmentVariablesWasm: e.EnvironmentVariables,
			EngineKeyImportModulesWasm:        e.ImportModules,
		},
	}
}

func WasmEngineFromEngineSpec(e EngineSpec) (WasmEngine, error) {
	if e.Type != EngineTypeWasm {
		return WasmEngine{}, fmt.Errorf("expected type %s got %s", EngineTypeWasm, e.Type)
	}
	if e.Params == nil {
		return WasmEngine{}, fmt.Errorf("engine params uninitialized")
	}
	var out WasmEngine
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StorageSpecUnmarshalledHookFunc(),
		Result:     &out,
	})
	if err != nil {
		return WasmEngine{}, err
	}
	if err := decoder.Decode(e.Params); err != nil {
		return WasmEngine{}, err
	}
	return out, nil
}

func StorageSpecUnmarshalledHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		from reflect.Type,
		to reflect.Type,
		data interface{}) (interface{}, error) {
		// TODO(forrest): [hack] this is unsafe, but I am unsure how else to handle this case.
		if to.String() != "model.StorageSourceType" {
			return data, nil
		}
		return ParseStorageSourceType(data.(string))
	}
}
