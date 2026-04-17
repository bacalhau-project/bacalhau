package transformer

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// LegacyWasmModuleTransformer converts jobs using the legacy WASM module format
// (where EntryModule and ImportModules are directly specified as InputSources) to the new format
// (using Target paths from InputSources).
type LegacyWasmModuleTransformer struct{}

func NewLegacyWasmModuleTransformer() *LegacyWasmModuleTransformer {
	return &LegacyWasmModuleTransformer{}
}

type legacySpec struct {
	EntryModule   *models.InputSource   `json:"EntryModule,omitempty"`
	ImportModules []*models.InputSource `json:"ImportModules,omitempty"`
	EntryPoint    string                `json:"EntryPoint,omitempty"`
	Parameters    []string              `json:"Parameters,omitempty"`
}

// Transform implements JobTransformer
func (t *LegacyWasmModuleTransformer) Transform(ctx context.Context, job *models.Job) error {
	if !job.Task().Engine.IsType(models.EngineWasm) {
		return nil // Not a WASM job, nothing to do
	}

	// First try to decode as new format
	spec, err := wasmmodels.DecodeSpec(job.Task().Engine)
	if err == nil && spec.EntryModule != "" {
		return nil // Already in new format, nothing to do
	}

	// Try to decode legacy format
	paramsBytes, err := json.Marshal(job.Task().Engine.Params)
	if err != nil {
		return err
	}

	var legacy legacySpec
	if err = json.Unmarshal(paramsBytes, &legacy); err != nil {
		return err
	}

	// If no legacy fields, nothing to do
	if legacy.EntryModule == nil && len(legacy.ImportModules) == 0 {
		return nil
	}

	// Create new spec with converted legacy fields
	newSpec := wasmmodels.EngineSpec{
		Entrypoint: legacy.EntryPoint,
		Parameters: legacy.Parameters,
	}

	// Convert entry module
	if legacy.EntryModule != nil {
		if legacy.EntryModule.Target == "" {
			legacy.EntryModule.Target = "/wasm/entry.wasm"
		}
		if err := legacy.EntryModule.Validate(); err != nil {
			return fmt.Errorf("invalid entry module: %w", err)
		}
		job.Task().InputSources = append(job.Task().InputSources, legacy.EntryModule)
		newSpec.EntryModule = legacy.EntryModule.Target
	}

	// Convert import modules
	for i, importModule := range legacy.ImportModules {
		if importModule != nil {
			if importModule.Target == "" {
				importModule.Target = filepath.Join("/wasm/imports", fmt.Sprintf("module_%d.wasm", i))
			}
			if err := importModule.Validate(); err != nil {
				return fmt.Errorf("invalid import module %d: %w", i, err)
			}
			job.Task().InputSources = append(job.Task().InputSources, importModule)
			newSpec.ImportModules = append(newSpec.ImportModules, importModule.Target)
		}
	}

	// Update the job's engine params
	job.Task().Engine.Params = newSpec.ToMap()
	return nil
}

// Compile-time check that LegacyWasmModuleTransformer implements JobTransformer
var _ JobTransformer = (*LegacyWasmModuleTransformer)(nil)
