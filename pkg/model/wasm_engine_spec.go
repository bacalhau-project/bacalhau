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

// WasmEngineBuilder is a struct used for constructing an EngineSpec object
// specifically for WebAssembly (Wasm) engines using the Builder pattern.
// It embeds an EngineBuilder object for handling the common builder methods.
type WasmEngineBuilder struct {
	eb EngineBuilder
}

// NewWasmEngineBuilder function initializes a new WasmEngineBuilder instance.
// It sets the engine type to engine.EngineWasm.String() and entry module as per the input argument.
func NewWasmEngineBuilder(entryModule StorageSpec) *WasmEngineBuilder {
	var eb EngineBuilder
	eb.WithType(EngineWasm.String())
	eb.WithParam(EngineKeyEntryModuleWasm, entryModule)
	return &WasmEngineBuilder{eb: eb}
}

// WithEntrypoint is a builder method that sets the WebAssembly engine's entrypoint.
// It returns the WasmEngineBuilder for further chaining of builder methods.
func (b *WasmEngineBuilder) WithEntrypoint(e string) *WasmEngineBuilder {
	b.eb.WithParam(EngineKeyEntrypointWasm, e)
	return b
}

// WithParameters is a builder method that sets the WebAssembly engine's parameters.
// It returns the WasmEngineBuilder for further chaining of builder methods.
func (b *WasmEngineBuilder) WithParameters(e ...string) *WasmEngineBuilder {
	b.eb.WithParam(EngineKeyParametersWasm, e)
	return b
}

// WithEnvironmentVariables is a builder method that sets the WebAssembly engine's environment variables.
// It returns the WasmEngineBuilder for further chaining of builder methods.
func (b *WasmEngineBuilder) WithEnvironmentVariables(e map[string]string) *WasmEngineBuilder {
	b.eb.WithParam(EngineKeyEnvironmentVariablesWasm, e)
	return b
}

// WithImportModules is a builder method that sets the WebAssembly engine's import modules.
// It returns the WasmEngineBuilder for further chaining of builder methods.
func (b *WasmEngineBuilder) WithImportModules(e ...StorageSpec) *WasmEngineBuilder {
	b.eb.WithParam(EngineKeyImportModulesWasm, e)
	return b
}

// Build method constructs the final EngineSpec object by calling the embedded EngineBuilder's Build method.
func (b *WasmEngineBuilder) Build() EngineSpec {
	return b.eb.Build()
}
