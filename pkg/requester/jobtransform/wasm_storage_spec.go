package jobtransform

import (
	"context"
	"fmt"

	wasm "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// NewWasmStorageSpecConverter returns a job transformer that converts the
// entry and import modules of wasm engine spec from the legacy storage spec to
// input source definitions.
func NewWasmStorageSpecConverter() PostTransformer {
	return func(ctx context.Context, j *models.Job) (modified bool, err error) {
		engineSpec := j.Task().Engine
		if engineSpec.IsType(models.EngineWasm) {
			wasmEngineSpec, err := wasm.DecodeLegacySpec(engineSpec)
			if err != nil {
				return false, fmt.Errorf("failed to decode wasm engine spec. %w", err)
			}
			engineSpec.Params = wasmEngineSpec.ToMap()
			return true, nil
		}
		return false, nil
	}
}
