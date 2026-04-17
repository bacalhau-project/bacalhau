package utils

import (
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// AllInputSources returns all storage types used by the job spec.
func AllInputSources(job *models.Job) []*models.InputSource {
	var inputSources []*models.InputSource
	for _, task := range job.Tasks {
		if task != nil && task.InputSources != nil {
			inputSources = append(inputSources, task.InputSources...)
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
