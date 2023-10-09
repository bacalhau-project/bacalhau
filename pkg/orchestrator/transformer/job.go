package transformer

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
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
		job.ID = idgen.NewJobID()
	}
	return nil
}

type JobDefaults struct {
	ExecutionTimeout time.Duration
}

// DefaultsApplier is a transformer that applies default values to the job.
func DefaultsApplier(defaults JobDefaults) Job {
	f := func(ctx context.Context, job *models.Job) error {
		for _, task := range job.Tasks {
			if task.Timeouts.GetExecutionTimeout() <= 0 {
				task.Timeouts.ExecutionTimeout = int64(defaults.ExecutionTimeout.Seconds())
			}
		}
		return nil
	}
	return JobFn(f)
}

// RequesterInfo is a transformer that sets the requester ID and public key in the job meta.
func RequesterInfo(requesterNodeID string, requesterPubKey model.PublicKey) Job {
	f := func(ctx context.Context, job *models.Job) error {
		job.Meta[models.MetaRequesterID] = requesterNodeID
		job.Meta[models.MetaRequesterPublicKey] = requesterPubKey.String()
		return nil
	}
	return JobFn(f)
}

// NameOptional is a transformer that sets the job name to the job ID if it is empty.
func NameOptional() Job {
	f := func(ctx context.Context, job *models.Job) error {
		if job.Name == "" {
			job.Name = job.ID
		}
		return nil
	}
	return JobFn(f)
}
