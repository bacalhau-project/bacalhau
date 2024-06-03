package transformer

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

// IDGenerator is a transformer that generates a new ID for the job if it is empty.
func IDGenerator(_ context.Context, job *models.Job) error {
	if job.ID == "" {
		job.ID = idgen.NewJobID()
	}
	return nil
}

type JobDefaults struct {
	ExecutionTimeout time.Duration
	QueueTimeout     time.Duration
}

// DefaultsApplier is a transformer that applies default values to the job.
func DefaultsApplier(defaults JobDefaults) JobTransformer {
	f := func(ctx context.Context, job *models.Job) error {
		for _, task := range job.Tasks {
			// only apply default execution timeout to non-long running jobs
			if !job.IsLongRunning() {
				if task.Timeouts.GetExecutionTimeout() <= 0 {
					task.Timeouts.ExecutionTimeout = int64(defaults.ExecutionTimeout.Seconds())
				}
			}

			// only apply default queue timeout to batch and service jobs
			if job.Type == models.JobTypeBatch || job.Type == models.JobTypeService {
				if task.Timeouts.GetQueueTimeout() <= 0 {
					task.Timeouts.QueueTimeout = int64(defaults.QueueTimeout.Seconds())
				}
			}
		}

		return nil
	}
	return JobFn(f)
}

// RequesterInfo is a transformer that sets the requester ID in the job meta.
func RequesterInfo(requesterNodeID string) JobTransformer {
	f := func(ctx context.Context, job *models.Job) error {
		job.Meta[models.MetaRequesterID] = requesterNodeID
		return nil
	}
	return JobFn(f)
}

// NameOptional is a transformer that sets the job name to the job ID if it is empty.
func NameOptional() JobTransformer {
	f := func(ctx context.Context, job *models.Job) error {
		if job.Name == "" {
			job.Name = job.ID
		}
		return nil
	}
	return JobFn(f)
}
