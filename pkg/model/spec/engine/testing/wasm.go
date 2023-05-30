package enginetesting

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
)

func WasmWithEntryModule(s spec.Storage) func(w *wasm.WasmEngineSpec) {
	return func(w *wasm.WasmEngineSpec) {
		w.EntryModule = s
	}
}

func WasmWithEntrypoint(e string) func(w *wasm.WasmEngineSpec) {
	return func(w *wasm.WasmEngineSpec) {
		w.EntryPoint = e
	}
}

func WasmWithParameters(p ...string) func(w *wasm.WasmEngineSpec) {
	return func(w *wasm.WasmEngineSpec) {
		w.Parameters = p
	}
}

func WasmWithEnvironmentVariables(e ...string) func(w *wasm.WasmEngineSpec) {
	return func(w *wasm.WasmEngineSpec) {
		w.EnvironmentVariables = e
	}
}

func WasmWithImportModules(i ...spec.Storage) func(w *wasm.WasmEngineSpec) {
	return func(w *wasm.WasmEngineSpec) {
		w.ImportModules = i
	}
}

func WasmMakeEngine(t testing.TB, opts ...func(engineSpec *wasm.WasmEngineSpec)) spec.Engine {
	w := &wasm.WasmEngineSpec{
		EntryModule:          spec.Storage{},
		EntryPoint:           "",
		Parameters:           nil,
		EnvironmentVariables: nil,
		ImportModules:        nil,
	}

	for _, opt := range opts {
		opt(w)
	}
	out, err := w.AsSpec()
	require.NoError(t, err)
	return out
}
