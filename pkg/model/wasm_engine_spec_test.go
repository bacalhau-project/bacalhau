//go:build unit || !integration

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWasmEngineBuilder(t *testing.T) {
	entryModule := StorageSpec{
		StorageSource: StorageSourceS3,
		Name:          "w2b",
	}
	builder := NewWasmEngineBuilder(entryModule)

	entrypoint := "_start"
	parameters := []string{"arg1", "arg2"}
	envVars := map[string]string{"VAR1": "value1", "VAR2": "value2"}
	importModules := []StorageSpec{{
		StorageSource: StorageSourceIPFS,
		Name:          "w3b",
	},
		{
			Name:          "w4b",
			StorageSource: StorageSourceIPFS,
		}}

	spec := builder.WithEntrypoint(entrypoint).
		WithParameters(parameters...).
		WithEnvironmentVariables(envVars).
		WithImportModules(importModules...).
		Build()

	require.Equal(t, EngineWasm.String(), spec.Type, "Engine type should be 'wasm'")

	assert.Equal(t, entryModule, spec.Params[EngineKeyEntryModuleWasm], "EntryModule should match")
	assert.Equal(t, entrypoint, spec.Params[EngineKeyEntrypointWasm], "Entrypoint should be equal to '%s', got '%s'", entrypoint, spec.Params[EngineKeyEntrypointWasm])
	assert.Equal(t, parameters, spec.Params[EngineKeyParametersWasm], "Parameters should be equal to '%v', got '%v'", parameters, spec.Params[EngineKeyParametersWasm])
	assert.Equal(t, envVars, spec.Params[EngineKeyEnvironmentVariablesWasm], "Environment variables should be equal to '%v', got '%v'", envVars, spec.Params[EngineKeyEnvironmentVariablesWasm])
	assert.Equal(t, importModules, spec.Params[EngineKeyImportModulesWasm], "ImportModules should be equal to '%v', got '%v'", importModules, spec.Params[EngineKeyImportModulesWasm])

	wasmEngine, err := DecodeEngineSpec[WasmEngineSpec](spec)
	require.NoError(t, err)

	assert.Equal(t, entryModule, wasmEngine.EntryModule, "EntryModule should match")
	assert.Equal(t, entrypoint, wasmEngine.Entrypoint, "Entrypoint should be equal to '%s', got '%s'", entrypoint, wasmEngine.Entrypoint)
	assert.Equal(t, parameters, wasmEngine.Parameters, "Parameters should be equal to '%v', got '%v'", parameters, wasmEngine.Parameters)
	assert.Equal(t, envVars, wasmEngine.EnvironmentVariables, "EnvironmentVariables should be equal to '%v', got '%v'", envVars, wasmEngine.EnvironmentVariables)
	assert.Equal(t, importModules, wasmEngine.ImportModules, "ImportModules should be equal to '%v', got '%v'", importModules, wasmEngine.ImportModules)
}
