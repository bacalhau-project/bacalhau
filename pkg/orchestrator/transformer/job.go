package transformer

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const (
	// Default scheduling timeout for jobs, in seconds
	DefaultSchedulingTimeout int64 = 60

	DefaultBaseRetryDelay int64 = 1

	DefaultMaximumRetryDelay int64 = 60

	DefaultRetryDelayGrowthFactor float64 = 2.0
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
}

// DefaultsApplier is a transformer that applies default values to the job.
func DefaultsApplier(defaults JobDefaults) JobTransformer {
	f := func(ctx context.Context, job *models.Job) error {
		// only apply default execution timeout to non-long running jobs
		if !job.IsLongRunning() {
			for _, task := range job.Tasks {
				if task.Timeouts.GetExecutionTimeout() <= 0 {
					task.Timeouts.ExecutionTimeout = int64(defaults.ExecutionTimeout.Seconds())
				}
			}
		}

		if job.ReschedulingPolicy.SchedulingTimeout == 0 {
			job.ReschedulingPolicy.SchedulingTimeout = DefaultSchedulingTimeout
		}

		if job.ReschedulingPolicy.BaseRetryDelay == 0 {
			job.ReschedulingPolicy.BaseRetryDelay = DefaultBaseRetryDelay
		}

		if job.ReschedulingPolicy.MaximumRetryDelay == 0 {
			job.ReschedulingPolicy.MaximumRetryDelay = DefaultMaximumRetryDelay
		}

		if job.ReschedulingPolicy.RetryDelayGrowthFactor < 1.0 {
			job.ReschedulingPolicy.RetryDelayGrowthFactor = DefaultRetryDelayGrowthFactor
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
