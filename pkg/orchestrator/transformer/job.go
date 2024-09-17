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
			if err := applyBatchJobDefaults(defaults.Batch, job); err != nil {
				return err
			}
		case models.JobTypeOps:
			if err := applyBatchJobDefaults(defaults.Ops, job); err != nil {
				return err
			}
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

func applyBatchJobDefaults(defaults types.BatchJobDefaultsConfig, job *models.Job) error {
	if job.Priority == 0 {
		job.Priority = defaults.Priority
	}
	for _, task := range job.Tasks {
		if err := applyBatchTaskDefaults(defaults.Task, task); err != nil {
			return err
		}
	}
	return nil
}

func applyBatchTaskDefaults(defaults types.BatchTaskDefaultConfig, task *models.Task) error {
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
	// if the user didn't provide a publisher, and a default transformer is set - use it.
	if task.Publisher.IsEmpty() && !defaults.Publisher.Config.IsEmpty() {
		task.Publisher = &defaults.Publisher.Config
	}
	if task.Timeouts.ExecutionTimeout <= 0 {
		task.Timeouts.ExecutionTimeout = int64(time.Duration(defaults.Timeouts.ExecutionTimeout).Seconds())
	}
	if task.Timeouts.TotalTimeout <= 0 {
		task.Timeouts.TotalTimeout = int64(time.Duration(defaults.Timeouts.TotalTimeout).Seconds())
	}

	return nil
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
}

// RequesterInfo is a transformer that sets the requester ID in the job meta.
func RequesterInfo(requesterNodeID string) JobTransformer {
	f := func(ctx context.Context, job *models.Job) error {
		job.Meta[models.MetaRequesterID] = requesterNodeID
		return nil
	}
	return JobFn(f)
}

func OrchestratorInstanceID(instanceID string) JobTransformer {
	f := func(ctx context.Context, job *models.Job) error {
		job.Meta[models.MetaServerInstanceID] = instanceID
		return nil
	}
	return JobFn(f)
}

func OrchestratorInstallationID(installationID string) JobTransformer {
	f := func(ctx context.Context, job *models.Job) error {
		job.Meta[models.MetaServerInstallationID] = installationID
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
