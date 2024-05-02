package models

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
	EntryModule *InputSource `json:"EntryModule,omitempty"`

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
	ImportModules []*InputSource `json:"ImportModules,omitempty"`
}

// WasmSpecConfigBuilder is a struct used for constructing an EngineSpec object
// specifically for WebAssembly (Wasm) engines using the Builder pattern.
// It embeds an EngineBuilder object for handling the common builder methods.
type WasmSpecConfigBuilder struct {
	sb *SpecConfig
}

// WasmSpecBuilder function initializes a new WasmSpecConfigBuilder instance.
// It sets the engine type to engine.EngineWasm.String() and entry module as per the input argument.
func WasmSpecBuilder(entryModule *InputSource) *WasmSpecConfigBuilder {
	sb := NewSpecConfig(EngineWasm)
	sb.WithParam(EngineKeyEntryModuleWasm, entryModule)
	return &WasmSpecConfigBuilder{sb: sb}
}

// WithEntrypoint is a builder method that sets the WebAssembly engine's entrypoint.
// It returns the WasmSpecConfigBuilder for further chaining of builder methods.
func (b *WasmSpecConfigBuilder) WithEntrypoint(e string) *WasmSpecConfigBuilder {
	b.sb.WithParam(EngineKeyEntrypointWasm, e)
	return b
}

// WithParameters is a builder method that sets the WebAssembly engine's parameters.
// It returns the WasmSpecConfigBuilder for further chaining of builder methods.
func (b *WasmSpecConfigBuilder) WithParameters(e ...string) *WasmSpecConfigBuilder {
	b.sb.WithParam(EngineKeyParametersWasm, e)
	return b
}

// WithEnvironmentVariables is a builder method that sets the WebAssembly engine's environment variables.
// It returns the WasmSpecConfigBuilder for further chaining of builder methods.
func (b *WasmSpecConfigBuilder) WithEnvironmentVariables(e map[string]string) *WasmSpecConfigBuilder {
	b.sb.WithParam(EngineKeyEnvironmentVariablesWasm, e)
	return b
}

// WithImportModules is a builder method that sets the WebAssembly engine's import modules.
// It returns the WasmSpecConfigBuilder for further chaining of builder methods.
func (b *WasmSpecConfigBuilder) WithImportModules(e ...*InputSource) *WasmSpecConfigBuilder {
	b.sb.WithParam(EngineKeyImportModulesWasm, e)
	return b
}

// Build method constructs the final EngineSpec object by calling the embedded EngineBuilder's Build method.
func (b *WasmSpecConfigBuilder) Build() (*SpecConfig, error) {
	b.sb.Normalize()
	if err := b.sb.Validate(); err != nil {
		return nil, err
	}
	return b.sb, nil
}
