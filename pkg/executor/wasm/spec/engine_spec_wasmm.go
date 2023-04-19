package spec

import (
	"bytes"
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
	EnvironmentVariables []KV `json:"EnvironmentVariables,omitempty"`

	// TODO #880: Other WASM modules whose exports will be available as imports
	// to the EntryModule.
	ImportModules []model.StorageSpec `json:"ImportModules,omitempty"`
}

type KV struct {
	Key   string
	Value string
}

func (ws *JobSpecWasm) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := ws.MarshalCBOR(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeJobSpecWasm(b []byte) (*JobSpecWasm, error) {
	var spec JobSpecWasm
	if err := spec.UnmarshalCBOR(bytes.NewReader(b)); err != nil {
		return nil, err
	}
	return &spec, nil
}

func (ws *JobSpecWasm) AsEngineSpec() model.EngineSpec {
	data, err := ws.Serialize()
	if err != nil {
		// TODO return to caller
		panic(err)
	}
	return model.EngineSpec{
		Type: WasmEngineType,
		Spec: data,
	}
}

func WithParameters(params ...string) func(wasm *JobSpecWasm) error {
	return func(wasm *JobSpecWasm) error {
		wasm.Parameters = params
		return nil
	}
}

func MutateWasmEngineSpec(e model.EngineSpec, mutate ...func(*JobSpecWasm) error) (model.EngineSpec, error) {
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

	return DecodeJobSpecWasm(e.Spec)
}
