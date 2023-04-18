package model

import (
	"strings"

	"github.com/ipld/go-ipld-prime/datamodel"
)

// TODO these are duplicated across the wasm executor package and here to avoid dep hell, need a better solution.
const (
	WasmEngineType             = EngineWasm
	WasmEngineEntryModuleKey   = "EntryModule"
	WasmEngineEntryPointKey    = "Entrypoint"
	WasmEngineParametersKey    = "Parameters"
	WasmEngineEnvVarKey        = "EnvironmentVariables"
	WasmEngineImportModulesKey = "ImportModules"
)

var _ JobType = (*WasmInputs)(nil)

type WasmInputs struct {
	Entrypoint string
	Parameters []string
	Modules    []Resource
	Mounts     IPLDMap[string, Resource] // Resource
	Outputs    IPLDMap[string, datamodel.Node]
	Env        IPLDMap[string, string]
}

func (wasm *WasmInputs) EngineSpec(_ string) (EngineSpec, error) {
	params := make(map[string]interface{})
	params[WasmEngineEntryPointKey] = wasm.Entrypoint
	params[WasmEngineParametersKey] = wasm.Parameters
	params[WasmEngineEnvVarKey] = wasm.Env.Values

	importModules := make([]StorageSpec, 0, len(wasm.Modules))
	for _, resource := range wasm.Modules {
		resource := resource
		importModules = append(importModules, parseStorageSource("", &resource))
	}
	params[WasmEngineImportModulesKey] = importModules

	return EngineSpec{
		Type: WasmEngineType,
		Spec: params,
	}, nil
}

func (wasm *WasmInputs) InputStorageSpecs(with string) ([]StorageSpec, error) {
	entryModule, err := parseResource(with)
	if err != nil {
		return nil, err
	}
	inputs := []StorageSpec{parseStorageSource("/job", entryModule)}

	inputData, err := parseInputs(wasm.Mounts)
	if err != nil {
		return nil, err
	}
	inputs = append(inputs, inputData...)
	return inputs, nil
}

func (wasm *WasmInputs) OutputStorageSpecs(_ string) ([]StorageSpec, error) {
	outputs := make([]StorageSpec, 0, len(wasm.Outputs.Values))
	for path := range wasm.Outputs.Values {
		outputs = append(outputs, StorageSpec{
			Path: path,
			Name: strings.Trim(path, "/"),
		})
	}
	return outputs, nil
}
