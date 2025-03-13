package utils

import (
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// AllInputSources returns all storage types used by the job spec.
func AllInputSources(job *models.Job) []*models.InputSource {
	inputSources := make([]*models.InputSource, 0, len(job.Tasks))
	for _, task := range job.Tasks {
		inputSources = append(inputSources, task.InputSources...)
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
