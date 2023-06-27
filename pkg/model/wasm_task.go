package model

import (
	"strings"

	"github.com/ipld/go-ipld-prime/datamodel"
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
func (wasm *WasmInputs) UnmarshalInto(with string, spec *Spec) error {

	entryModule, err := parseResource(with)
	if err != nil {
		return err
	}
	spec.Inputs = []StorageSpec{parseStorageSource("/job", entryModule)}

	var importModules []StorageSpec
	for _, resource := range wasm.Modules {
		// TODO(forrest): [correctness] this is correct but easily bug prone: https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
		resource := resource
		importModules = append(importModules, parseStorageSource("", &resource))
	}

	// TODO(forrest): [logic change] I am not sure this is the correct way to set an entryModule, previously we never set it, attempting to now.
	spec.EngineSpec = NewWasmEngineSpec(parseStorageSource("", entryModule), wasm.Entrypoint, wasm.Parameters, wasm.Env.Values, importModules)
	spec.EngineDeprecated = EngineWasm

	inputData, err := parseInputs(wasm.Mounts)
	if err != nil {
		return err
	}
	spec.Inputs = append(spec.Inputs, inputData...)

	spec.Outputs = []StorageSpec{}
	for path := range wasm.Outputs.Values {
		spec.Outputs = append(spec.Outputs, StorageSpec{
			Path: path,
			Name: strings.Trim(path, "/"),
		})
	}

	return nil
}
