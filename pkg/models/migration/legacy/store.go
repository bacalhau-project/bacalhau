package legacy

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// GetJobState is a helper function that returns the job state for a given job ID
func GetJobState(ctx context.Context, jobStore jobstore.Store, jobID string) (model.JobState, error) {
	job, err := jobStore.GetJob(ctx, jobID)
	if err != nil {
		return model.JobState{}, err
	}
	executions, err := jobStore.GetExecutions(ctx, jobID)
	if err != nil {
		return model.JobState{}, err
	}

	jobState, err := ToLegacyJobStatus(job, executions)
	if err != nil {
		return model.JobState{}, err
	}

	return *jobState, nil
}

func NewStateResolver(jobStore jobstore.Store) *job.StateResolver {
	return job.NewStateResolver(
		func(ctx context.Context, jobID string) (model.Job, error) {
			return GetJob(ctx, jobStore, jobID)
		},
		func(ctx context.Context, jobID string) (model.JobState, error) {
			return GetJobState(ctx, jobStore, jobID)
		},
	)
}

func GetJob(ctx context.Context, jobStore jobstore.Store, jobID string) (model.Job, error) {
	newJob, err := jobStore.GetJob(ctx, jobID)
	if err != nil {
		return model.Job{}, err
	}
	legacyJob, err := ToLegacyJob(&newJob)
	if err != nil {
		return model.Job{}, err
	}
	return *legacyJob, nil
}
