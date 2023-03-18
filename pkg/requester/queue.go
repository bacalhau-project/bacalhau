package requester

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/pkg/errors"
)

type queue struct {
	scheduler Scheduler
	store     jobstore.Store
}

func NewQueue(store jobstore.Store, scheduler Scheduler) Queue {
	return &queue{
		scheduler: scheduler,
		store:     store,
	}
}

func (q *queue) EnqueueJob(ctx context.Context, job model.Job) error {
	return q.store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
		JobID: job.Metadata.ID,
		Condition: jobstore.UpdateJobCondition{
			ExpectedState: model.JobStateNew,
		},
		NewState: model.JobStateQueued,
	})
}

func (q *queue) StartJob(ctx context.Context, req StartJobRequest) error {
	err := q.store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
		JobID: req.Job.Metadata.ID,
		Condition: jobstore.UpdateJobCondition{
			ExpectedState: model.JobStateQueued,
		},
		NewState: model.JobStateNew,
	})
	if err != nil {
		return err
	}

	return q.scheduler.StartJob(ctx, req)
}

func (q *queue) CancelJob(ctx context.Context, req CancelJobRequest) (CancelJobResult, error) {
	err := q.store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
		JobID: req.JobID,
		Condition: jobstore.UpdateJobCondition{
			ExpectedState: model.JobStateQueued,
		},
		NewState: model.JobStateCancelled,
		Comment:  req.Reason,
	})
	var invalidJobErr jobstore.ErrInvalidJobState
	if err != nil && errors.As(err, &invalidJobErr) {
		return q.scheduler.CancelJob(ctx, req)
	}
	return CancelJobResult{}, err
}
