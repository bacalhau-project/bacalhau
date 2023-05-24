package model

import (
	"strings"

	"github.com/ipld/go-ipld-prime/datamodel"

	"github.com/bacalhau-project/bacalhau/pkg/model/specs/engine/wasm"
)

type WasmInputs struct {
	Entrypoint string
	Parameters []string
	Modules    []Resource
	Mounts     IPLDMap[string, Resource] // Resource
	Outputs    IPLDMap[string, datamodel.Node]
	Env        IPLDMap[string, string]
}

var _ JobType = (*WasmInputs)(nil)

// UnmarshalInto implements taskUnmarshal
func (w *WasmInputs) UnmarshalInto(with string, spec *Spec) error {
	wasmEngine := wasm.WasmEngineSpec{
		EntryPoint:           w.Entrypoint,
		Parameters:           w.Parameters,
		EnvironmentVariables: FlattenIPLDMap(w.Env),
		ImportModules:        nil,
	}

	entryModule, err := parseResource(with)
	if err != nil {
		return err
	}
	spec.Inputs = []StorageSpec{parseStorageSource("/job", entryModule)}

	for _, resource := range w.Modules {
		_ = resource
		panic("TODO")
		//spec.Wasm.ImportModules = append(spec.Wasm.ImportModules, parseStorageSource("", &resource))
	}

	spec.Engine, err = wasmEngine.AsSpec()
	if err != nil {
		return err
	}

	inputData, err := parseInputs(w.Mounts)
	if err != nil {
		return err
	}
	spec.Inputs = append(spec.Inputs, inputData...)

	spec.Outputs = []StorageSpec{}
	for path := range w.Outputs.Values {
		spec.Outputs = append(spec.Outputs, StorageSpec{
			Path: path,
			Name: strings.Trim(path, "/"),
		})
	}

	return nil
}
