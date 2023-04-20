package model

import (
	"encoding/json"
	"fmt"
)

// TODO these are duplicated across the wasm executor package and here to avoid dep hell, need a better solution.
const (
	WasmEngineType = 3
)

// JobSpecWasm describes a raw WASM job.
type JobSpecWasm struct {
	// The module that contains the WASM code to start running.
	EntryModule StorageSpec `json:"EntryModule,omitempty"`

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
	ImportModules []StorageSpec `json:"ImportModules,omitempty"`
}

func (ws *JobSpecWasm) AsEngineSpec() EngineSpec {
	data, err := json.Marshal(ws)
	if err != nil {
		panic(err)
	}
	return EngineSpec{
		Type: WasmEngineType,
		Spec: data,
	}
}

func AsJobSpecWasm(e EngineSpec) (*JobSpecWasm, error) {
	if e.Type != WasmEngineType {
		return nil, fmt.Errorf("EngineSpec is Type %s, expected %d", e.Type, WasmEngineType)
	}

	if e.Spec == nil {
		return nil, fmt.Errorf("EngineSpec is uninitalized")
	}

	out := new(JobSpecWasm)
	if err := json.Unmarshal(e.Spec, out); err != nil {
		return nil, err
	}
	return out, nil
}

func WithParameters(params ...string) func(wasm *JobSpecWasm) error {
	return func(wasm *JobSpecWasm) error {
		wasm.Parameters = params
		return nil
	}
}

func MutateWasmEngineSpec(e EngineSpec, mutate ...func(*JobSpecWasm) error) (EngineSpec, error) {
	wasmSpec, err := AsJobSpecWasm(e)
	if err != nil {
		return EngineSpec{}, err
	}

	for _, m := range mutate {
		if err := m(wasmSpec); err != nil {
			return EngineSpec{}, err
		}
	}
	return wasmSpec.AsEngineSpec(), nil
}
