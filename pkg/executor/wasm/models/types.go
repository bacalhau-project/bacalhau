package models

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fatih/structs"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// EngineSpec contains necessary parameters to execute a wasm job.
type EngineSpec struct {
	// EntryModule is the target path of the input source containing the WASM code to start running.
	// This target must match an InputSource target in the job spec.
	EntryModule string `json:"EntryModule"`

	// Entrypoint is the name of the function in the EntryModule to call to run the job.
	// For WASI jobs, this will should be `_start`, but jobs can choose to call other WASM functions instead.
	// Entrypoint must be a zero-parameter zero-result function.
	Entrypoint string `json:"Entrypoint,omitempty"`

	// Parameters contains arguments supplied to the program (i.e. as ARGV).
	Parameters []string `json:"Parameters,omitempty"`

	// ImportModules is a slice of target paths for WASM modules whose exports will be available as imports
	// to the EntryModule. These targets must match InputSource targets in the job spec.
	ImportModules []string `json:"ImportModules,omitempty"`
}

func (c EngineSpec) Validate() error {
	if c.EntryModule == "" {
		return errors.New("invalid wasm engine entry module. target path cannot be empty")
	}
	return nil
}

func (c EngineSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (EngineSpec, error) {
	if !spec.IsType(models.EngineWasm) {
		return EngineSpec{}, errors.New("invalid wasm engine type. expected " + models.EngineWasm + ", but received: " + spec.Type)
	}

	inputParams := spec.Params
	if inputParams == nil {
		return EngineSpec{}, errors.New("invalid wasm engine params. cannot be nil")
	}

	paramsBytes, err := json.Marshal(inputParams)
	if err != nil {
		return EngineSpec{}, fmt.Errorf("failed to encode wasm engine specs. %w", err)
	}

	var c *EngineSpec
	err = json.Unmarshal(paramsBytes, &c)
	if err != nil {
		return EngineSpec{}, err
	}

	return *c, c.Validate()
}

type WasmEngineBuilder struct {
	spec *EngineSpec
}

func NewWasmEngineBuilder(entryModuleAlias string) *WasmEngineBuilder {
	spec := &EngineSpec{
		EntryModule: entryModuleAlias,
	}

	return &WasmEngineBuilder{spec: spec}
}

func (b *WasmEngineBuilder) WithEntrypoint(e string) *WasmEngineBuilder {
	b.spec.Entrypoint = e
	return b
}

func (b *WasmEngineBuilder) WithParameters(e ...string) *WasmEngineBuilder {
	b.spec.Parameters = e
	return b
}

func (b *WasmEngineBuilder) WithImportModules(e []string) *WasmEngineBuilder {
	b.spec.ImportModules = e
	return b
}

func (b *WasmEngineBuilder) Build() (*models.SpecConfig, error) {
	if err := b.spec.Validate(); err != nil {
		return nil, err
	}
	return &models.SpecConfig{
		Type:   models.EngineWasm,
		Params: b.spec.ToMap(),
	}, nil
}

func (b *WasmEngineBuilder) MustBuild() *models.SpecConfig {
	spec, err := b.Build()
	if err != nil {
		panic(err)
	}
	return spec
}
