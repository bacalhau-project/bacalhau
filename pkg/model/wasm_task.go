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
	entryModuleStorageSpec := parseStorageSource("/job", entryModule)
	spec.Inputs = []StorageSpec{entryModuleStorageSpec}

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

	importModules := make([]StorageSpec, 0, len(wasm.Modules))
	for _, resource := range wasm.Modules {
		resource := resource
		importModules = append(importModules, parseStorageSource("", &resource))
	}

	spec.EngineSpec = NewWasmEngineBuilder(entryModuleStorageSpec).
		WithEntrypoint(wasm.Entrypoint).
		WithParameters(wasm.Parameters...).
		WithEnvironmentVariables(wasm.Env.Values).
		WithImportModules(importModules...).
		Build()

	return nil
}
