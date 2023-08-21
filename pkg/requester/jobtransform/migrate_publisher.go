package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// Maintains backward compatibility for jobs that were defined with Publisher and PublisherSpec
func NewPublisherMigrator() Transformer {
	return func(ctx context.Context, job *model.Job) (modified bool, err error) {
		if model.IsValidPublisher(job.Spec.PublisherSpec.Type) {
			job.Spec.Publisher = job.Spec.PublisherSpec.Type
		} else {
			job.Spec.PublisherSpec.Type = job.Spec.Publisher
		}
		return true, nil
	}
}
