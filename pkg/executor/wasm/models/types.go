package models

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

const (
	EngineKeyEntryModuleWasm          = "EntryModule"
	EngineKeyEntrypointWasm           = "Entrypoint"
	EngineKeyParametersWasm           = "Parameters"
	EngineKeyEnvironmentVariablesWasm = "EnvironmentVariables"
	EngineKeyImportModulesWasm        = "ImportModules"
)

// EngineSpec contains necessary parameters to execute a wasm job.
type EngineSpec struct {
	// EntryModule is a Spec containing the WASM code to start running.
	EntryModule *models.InputSource `json:"EntryModule,omitempty"`

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
	ImportModules []*models.InputSource `json:"ImportModules,omitempty"`
}

func (c EngineSpec) Validate() error {
	if c.EntryModule == nil {
		return errors.New("invalid wasm engine entry module. cannot be nil")
	}
	return nil
}

func (c EngineSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (EngineSpec, error) {
	if spec.Type != models.EngineWasm {
		return EngineSpec{}, errors.New("invalid wasm engine type. expected " + models.EngineWasm + ", but received: " + spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return EngineSpec{}, errors.New("invalid wasm engine params. cannot be nil")
	}

	var c EngineSpec
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}

type EngineArguments struct {
	EntryPoint           string
	Parameters           []string
	EnvironmentVariables map[string]string
	EntryModule          storage.PreparedStorage
	ImportModules        []storage.PreparedStorage
}

func DecodeArguments(spec *models.SpecConfig) (*EngineArguments, error) {
	if spec.Type != models.EngineWasm {
		return nil, errors.New("invalid wasm engine type. expected " + models.EngineWasm + ", but received: " + spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return nil, errors.New("invalid wasm engine params. cannot be nil")
	}

	var c *EngineArguments
	if err := mapstructure.Decode(spec.Params, c); err != nil {
		return c, err
	}

	return c, nil
}

// WasmEngineBuilder is a struct used for constructing an EngineSpec object
// specifically for WebAssembly (Wasm) engines using the Builder pattern.
// It embeds an EngineBuilder object for handling the common builder methods.
type WasmEngineBuilder struct {
	eb *models.SpecConfig
}

// NewWasmEngineBuilder function initializes a new WasmEngineBuilder instance.
// It sets the engine type to engine.EngineWasm.String() and entry module as per the input argument.
func NewWasmEngineBuilder(entryModule storage.PreparedStorage) *WasmEngineBuilder {
	eb := models.NewSpecConfig(models.EngineWasm)
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
func (b *WasmEngineBuilder) WithImportModules(e ...storage.PreparedStorage) *WasmEngineBuilder {
	b.eb.WithParam(EngineKeyImportModulesWasm, e)
	return b
}

// Build method constructs the final SpecConfig object by calling the embedded EngineBuilder's Build method.
func (b *WasmEngineBuilder) Build() *models.SpecConfig {
	return b.eb
}
