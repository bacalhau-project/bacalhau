package utils

import (
	wasm "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// AllInputSources returns all storage types used by the job spec.
// TODO: this is a temporary hack to get the storage type from the job spec, including remote sources
// defined inside the wasm spec. Long term solution to move remote sources from wasm spec to outer task spec.
func AllInputSources(job *models.Job) []*models.InputSource {
	inputSources := make([]*models.InputSource, 0, len(job.Tasks))
	for _, task := range job.Tasks {
		inputSources = append(inputSources, task.InputSources...)
		if task.Engine.Type == models.EngineWasm {
			wasmEngineSpec, err := wasm.DecodeSpec(task.Engine)
			if err != nil {
				log.Error().Err(err).Msgf("failed to decode wasm engine spec %+v", task.Engine)
			}
			inputSources = append(inputSources, wasmEngineSpec.EntryModule)
			inputSources = append(inputSources, wasmEngineSpec.ImportModules...)
		}
	}
	return inputSources
}

// AllInputSourcesTypes returns all storage types used by the job spec.
func AllInputSourcesTypes(job *models.Job) []string {
	inputTypes := make(map[string]struct{})
	for _, input := range AllInputSources(job) {
		inputTypes[input.Source.Type] = struct{}{}
	}
	return lo.Keys(inputTypes)
}
