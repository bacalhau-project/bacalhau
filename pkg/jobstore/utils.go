package jobstore

import (
	"context"

	"github.com/rs/zerolog/log"

	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func GetStateResolver(db Store) *jobutils.StateResolver {
	return jobutils.NewStateResolver(
		db.GetJob,
		db.GetJobState,
	)
}

// StopJob a helper function to fail a job and all its executions.
func StopJob(ctx context.Context, db Store, jobID string, reason string, userRequested bool) ([]model.ExecutionState, error) {
	// update job state
	newJobState := model.JobStateError
	unexpectedJobState := model.JobStateCancelled
	if userRequested {
		newJobState = model.JobStateCancelled
		unexpectedJobState = model.JobStateError
	}
	err := db.UpdateJobState(ctx, UpdateJobStateRequest{
		JobID: jobID,
		Condition: UpdateJobCondition{
			UnexpectedStates: []model.JobStateType{
				model.JobStateCompleted,
				unexpectedJobState,
			},
		},
		NewState: newJobState,
		Comment:  reason,
	})
	if err != nil {
		return nil, err
	}

	// update execution state
	jobState, err := db.GetJobState(ctx, jobID)
	if err != nil {
		return nil, err
	}

	cancelledExecutions := make([]model.ExecutionState, 0)
	for _, execution := range jobState.Executions {
		if !execution.State.IsTerminal() {
			err = db.UpdateExecutionState(ctx, execution.ID(), UpdateExecutionStateRequest{
				Condition: UpdateExecutionCondition{
					UnexpectedStates: []model.ExecutionStateType{
						model.ExecutionStateFailed,
						model.ExecutionStateCompleted,
					},
				},
				NewState: model.ExecutionStateCanceled,
				Comment:  reason,
			})
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msgf("failed to update execution state to Canceled. %s", execution)
			} else {
				cancelledExecutions = append(cancelledExecutions, execution)
			}
		}
	}
	return cancelledExecutions, nil
}
