package transformer

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
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

// DefaultsApplier is a transformer that applies default values to the job.
func DefaultsApplier(defaults types.JobDefaults) JobTransformer {
	f := func(ctx context.Context, job *models.Job) error {
		switch job.Type {
		case models.JobTypeBatch:
			applyBatchJobDefaults(defaults.Batch, job)
		case models.JobTypeOps:
			applyBatchJobDefaults(defaults.Ops, job)
		case models.JobTypeService:
			applyLongRunningJobDefaults(defaults.Service, job)
		case models.JobTypeDaemon:
			applyLongRunningJobDefaults(defaults.Daemon, job)
		default:
			return fmt.Errorf("unknown job type: %s", job.Type)
		}

		return nil
	}
	return JobFn(f)
}

func applyBatchJobDefaults(defaults types.BatchJobDefaultsConfig, job *models.Job) {
	if job.Priority == 0 {
		job.Priority = defaults.Priority
	}
	for _, task := range job.Tasks {
		applyBatchTaskDefaults(defaults.Task, task)
	}
}

func applyBatchTaskDefaults(defaults types.BatchTaskDefaultConfig, task *models.Task) {
	if task.ResourcesConfig.CPU == "" {
		task.ResourcesConfig.CPU = defaults.Resources.CPU
	}
	if task.ResourcesConfig.Memory == "" {
		task.ResourcesConfig.Memory = defaults.Resources.Memory
	}
	if task.ResourcesConfig.Disk == "" {
		task.ResourcesConfig.Disk = defaults.Resources.Disk
	}
	if task.ResourcesConfig.GPU == "" {
		task.ResourcesConfig.GPU = defaults.Resources.GPU
	}
	if task.Publisher.IsEmpty() {
		task.Publisher.Type = defaults.Publisher.Type
	}
	if task.Timeouts.ExecutionTimeout <= 0 {
		task.Timeouts.ExecutionTimeout = int64(time.Duration(defaults.Timeouts.ExecutionTimeout).Seconds())
	}
	if task.Timeouts.TotalTimeout <= 0 {
		task.Timeouts.TotalTimeout = int64(time.Duration(defaults.Timeouts.TotalTimeout).Seconds())
	}
}

func applyLongRunningJobDefaults(defaults types.LongRunningJobDefaultsConfig, job *models.Job) {
	if job.Priority == 0 {
		job.Priority = defaults.Priority
	}
	for _, task := range job.Tasks {
		applyLongRunningTaskDefaults(defaults.Task, task)
	}
}

func applyLongRunningTaskDefaults(defaults types.LongRunningTaskDefaultConfig, task *models.Task) {
	if task.ResourcesConfig.CPU == "" {
		task.ResourcesConfig.CPU = defaults.Resources.CPU
	}
	if task.ResourcesConfig.Memory == "" {
		task.ResourcesConfig.Memory = defaults.Resources.Memory
	}
	if task.ResourcesConfig.Disk == "" {
		task.ResourcesConfig.Disk = defaults.Resources.Disk
	}
	if task.ResourcesConfig.GPU == "" {
		task.ResourcesConfig.GPU = defaults.Resources.GPU
	}
	if task.Publisher.IsEmpty() {
		task.Publisher.Type = defaults.Publisher.Type
	}
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
