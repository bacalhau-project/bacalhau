package jobtransform

import (
	"context"

	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

// Maintains backward compatibility for jobs that were defined with Publisher and PublisherSpec
func NewPublisherMigrator(defaultPublisher string) Transformer {
	var err error
	var pubSpec model.PublisherSpec

	if defaultPublisher != "" {
		pubSpec, err = legacy_job.RawPublisherStringToPublisherSpec(defaultPublisher)
		if err != nil {
			log.Error().
				Err(err).
				Str("Publisher", defaultPublisher).
				Msg("Failed to parse default publisher: using noop publisher instead")
		}
	}

	return func(ctx context.Context, job *model.Job) (modified bool, err error) {
		// Only set the default publisher if we have one defined and we are presented
		// with a noop publisher (which is used when the user does not want to publish)
		if job.Spec.PublisherSpec.Type == model.PublisherNoop && pubSpec.Type != model.PublisherNoop {
			job.Spec.PublisherSpec = pubSpec
		}

		if model.IsValidPublisher(job.Spec.PublisherSpec.Type) {
			//nolint:staticcheck
			job.Spec.Publisher = job.Spec.PublisherSpec.Type
		} else {
			//nolint:staticcheck
			job.Spec.PublisherSpec.Type = job.Spec.Publisher
		}
		return true, nil
	}
}
