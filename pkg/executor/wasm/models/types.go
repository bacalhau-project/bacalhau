package models

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/fatih/structs"
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
	if c.EntryModule.Source == nil {
		return errors.New("invalid wasm engine entry module. source cannot be nil")
	}
	return nil
}

// ToArguments returns EngineArguments from the spec
func (c EngineSpec) ToArguments(entryModule storage.PreparedStorage, importModules ...storage.PreparedStorage) EngineArguments {
	return EngineArguments{
		EntryModule:          entryModule,
		EntryPoint:           c.Entrypoint,
		Parameters:           c.Parameters,
		EnvironmentVariables: c.EnvironmentVariables,
		ImportModules:        importModules,
	}
}

func (c EngineSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

// legacyEngineSpec is first used to decode the passed spec with legacy model.StorageSpec,
// and then converted to EngineSpec with models.InputSource
type legacyEngineSpec struct {
	EntryModule          model.StorageSpec
	Entrypoint           string
	Parameters           []string
	EnvironmentVariables map[string]string
	ImportModules        []model.StorageSpec
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

func DecodeLegacySpec(spec *models.SpecConfig) (EngineSpec, error) {
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

	var c *legacyEngineSpec
	err = json.Unmarshal(paramsBytes, &c)
	if err != nil {
		return EngineSpec{}, err
	}

	entryModule, err := legacy.FromLegacyStorageSpecToInputSource(c.EntryModule)
	if err != nil {
		return EngineSpec{}, err
	}

	importModules := make([]*models.InputSource, 0, len(c.ImportModules))
	for _, module := range c.ImportModules {
		newModule, err := legacy.FromLegacyStorageSpecToInputSource(module)
		if err != nil {
			return EngineSpec{}, err
		}
		importModules = append(importModules, newModule)
	}

	engineSpec := EngineSpec{
		EntryModule:          entryModule,
		Entrypoint:           c.Entrypoint,
		Parameters:           c.Parameters,
		EnvironmentVariables: c.EnvironmentVariables,
		ImportModules:        importModules,
	}
	return engineSpec, engineSpec.Validate()
}

// EngineArguments is used to pass pre-processed engine specs to the executor.
// Currently used to pre-fetch entry and import modules remote resources by the compute
// node before triggering the executor.
// TODO: deprecate these arguments once we move remote resources from the engine spec to
// the upper layer
type EngineArguments struct {
	EntryPoint           string
	Parameters           []string
	EnvironmentVariables map[string]string
	EntryModule          storage.PreparedStorage
	ImportModules        []storage.PreparedStorage
}

func (c EngineArguments) Validate() error {
	if (c.EntryModule.InputSource == models.InputSource{}) {
		return errors.New("invalid wasm engine entry module. cannot be empty")
	}
	if c.EntryModule.InputSource.Source == nil {
		return errors.New("invalid wasm engine entry module. source cannot be nil")
	}
	return nil
}

func (c EngineArguments) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeArguments(spec *models.SpecConfig) (*EngineArguments, error) {
	if !spec.IsType(models.EngineWasm) {
		return nil, errors.New("invalid wasm engine type. expected " + models.EngineWasm + ", but received: " + spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return nil, errors.New("invalid wasm engine params. cannot be nil")
	}

	paramsBytes, err := json.Marshal(inputParams)
	if err != nil {
		return nil, fmt.Errorf("failed to encode wasm engine specs. %w", err)
	}

	var c *EngineArguments
	err = json.Unmarshal(paramsBytes, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to decode wasm engine specs. %w", err)
	}
	return c, c.Validate()
}
