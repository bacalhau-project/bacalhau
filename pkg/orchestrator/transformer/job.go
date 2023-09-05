package transformer

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/google/uuid"
)

// ChainedJobTransformer is a slice of Transformers that runs in sequence
type ChainedJobTransformer []Job

// Transform runs all transformers in sequence.
func (ct ChainedJobTransformer) Transform(ctx context.Context, job *models.Job) error {
	for _, t := range ct {
		if err := t.Transform(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

// IDGenerator is a transformer that generates a new ID for the job if it is empty.
func IDGenerator(_ context.Context, job *models.Job) error {
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	return nil
}
