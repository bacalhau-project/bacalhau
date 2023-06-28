package model

import (
	"encoding/json"
	"fmt"
)

const (
	EngineTypeWasm                    = "wasm"
	EngineKeyEntryModuleWasm          = "entrymodule"
	EngineKeyEntrypointWasm           = "entrypoint"
	EngineKeyParametersWasm           = "parameters"
	EngineKeyEnvironmentVariablesWasm = "environmentvariables"
	EngineKeyImportModulesWasm        = "importmodules"
)

// NewWasmEngineSpec returns an EngineSpec of type EngineTypeWasm with the provided arguments as EngineSpec.Params.
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

// WasmEngineSpec contains necessary parameters to execute a wasm job.
type WasmEngineSpec struct {
	// EntryModule is a Spec containing the WASM code to start running.
	EntryModule StorageSpec `json:"EntryModule,omitempty"`

	// Entrypoint is the name of the function in the EntryModule to call to run the job.
	// For WASI jobs, this will should be `_start`, but jobs can choose to call other WASM functions instead.
	// Entrypoint must be a zero-parameter zero-result function.
	Entrypoint string `json:"EntryPoint,omitempty"`

	// Parameters contains arguments supplied to the program (i.e. as ARGV).
	Parameters []string `json:"Parameters,omitempty"`

	// EnvironmentVariables contains variables available in the environment of the running program.
	EnvironmentVariables map[string]string `json:"EnvironmentVariables,omitempty"`

	// ImportModules is a slice of StorageSpec's containing WASM modules whose exports will be available as imports
	// to the EntryModule.
	ImportModules []StorageSpec `json:"ImportModules,omitempty"`
}

// AsEngineSpec returns a WasmEngineSpec as an EngineSpec.
func (e WasmEngineSpec) AsEngineSpec() EngineSpec {
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

// WasmEngineSpecFromEngineSpec decodes a WasmEngineSpec from an EngineSpec.
// This method will return an error if:
// - The EngineSpec argument is not of type EngineTypeWasm.
// - The EngineSpec.Params are nil.
// - The EngineSpec.Params cannot be marshaled to json bytes.
// - The EngineSpec.Params cannot be unmarshalled to a WasmEngineSpec.
func WasmEngineSpecFromEngineSpec(e EngineSpec) (WasmEngineSpec, error) {
	if e.Type != EngineTypeWasm {
		return WasmEngineSpec{}, fmt.Errorf("expected type %s got %s", EngineTypeWasm, e.Type)
	}
	if e.Params == nil {
		return WasmEngineSpec{}, fmt.Errorf("engine params uninitialized")
	}
	// NB(forrest): we rely on go's json marshaller to handle the conversion of e.Params map[string]interface{} to the
	// typed structure WasmEngineSpec.
	eb, err := json.Marshal(e.Params)
	if err != nil {
		return WasmEngineSpec{}, nil
	}
	var out WasmEngineSpec
	if err := json.Unmarshal(eb, &out); err != nil {
		return WasmEngineSpec{}, err
	}
	return out, nil
}
