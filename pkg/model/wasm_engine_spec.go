package model

const (
	EngineKeyEntryModuleWasm          = "EntryModule"
	EngineKeyEntrypointWasm           = "Entrypoint"
	EngineKeyParametersWasm           = "Parameters"
	EngineKeyEnvironmentVariablesWasm = "EnvironmentVariables"
	EngineKeyImportModulesWasm        = "ImportModules"
)

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

type WasmEngineBuilder struct {
	eb EngineBuilder
}

func NewWasmEngineBuilder(entryModule StorageSpec) *WasmEngineBuilder {
	var eb EngineBuilder
	eb.WithType(EngineWasm.String())
	eb.WithParam(EngineKeyEntryModuleWasm, entryModule)
	return &WasmEngineBuilder{eb: eb}
}

func (b *WasmEngineBuilder) WithEntrypoint(e string) *WasmEngineBuilder {
	b.eb.WithParam(EngineKeyEntrypointWasm, e)
	return b
}

func (b *WasmEngineBuilder) WithParameters(e ...string) *WasmEngineBuilder {
	b.eb.WithParam(EngineKeyParametersWasm, e)
	return b
}

func (b *WasmEngineBuilder) WithEnvironmentVariables(e map[string]string) *WasmEngineBuilder {
	b.eb.WithParam(EngineKeyEnvironmentVariablesWasm, e)
	return b
}

func (b *WasmEngineBuilder) WithImportModules(e ...StorageSpec) *WasmEngineBuilder {
	b.eb.WithParam(EngineKeyImportModulesWasm, e)
	return b
}

func (b *WasmEngineBuilder) Build() EngineSpec {
	return b.eb.Build()
}
