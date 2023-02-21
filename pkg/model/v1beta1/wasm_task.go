package v1beta1

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
	spec.Engine = EngineWasm
	spec.Wasm = JobSpecWasm{
		EntryPoint:           wasm.Entrypoint,
		Parameters:           wasm.Parameters,
		EnvironmentVariables: wasm.Env.Values,
		ImportModules:        []StorageSpec{},
	}

	entryModule, err := parseResource(with)
	if err != nil {
		return err
	}
	spec.Contexts = []StorageSpec{parseStorageSource("/job", entryModule)}

	for _, resource := range wasm.Modules {
		resource := resource
		spec.Wasm.ImportModules = append(spec.Wasm.ImportModules, parseStorageSource("", &resource))
	}

	inputData, err := parseInputs(wasm.Mounts)
	if err != nil {
		return err
	}
	spec.Inputs = inputData

	spec.Outputs = []StorageSpec{}
	for path := range wasm.Outputs.Values {
		spec.Outputs = append(spec.Outputs, StorageSpec{
			Path: path,
			Name: strings.Trim(path, "/"),
		})
	}

	return nil
}
