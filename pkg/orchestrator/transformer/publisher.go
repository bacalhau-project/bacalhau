package transformer

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func DefaultPublisher(publisherConfig *models.SpecConfig) JobTransformer {
	f := func(ctx context.Context, job *models.Job) error {
		for i := range job.Tasks {
			task := job.Tasks[i]
			if task.Publisher == nil || task.Publisher.Type == "" {
				task.Publisher = publisherConfig
			}
		}

		return nil
	}

	return JobFn(f)
}
