package jobstore

import (
	"context"

	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

func GetStateResolver(db Store) *jobutils.StateResolver {
	return jobutils.NewStateResolver(
		db.GetJob,
		db.GetJobState,
	)
}

// CompleteShard a helper function to complete a shard, and update the job state if all other shards are completed.
func CompleteShard(ctx context.Context, db Store, shardID model.ShardID) error {
	shardState, err := db.GetShardState(ctx, shardID)
	if err != nil {
		return err
	}
	if shardState.State == model.ShardStateCompleted {
		return nil
	}
	// update shard state
	err = db.UpdateShardState(ctx, UpdateShardStateRequest{
		ShardID:  shardID,
		NewState: model.ShardStateCompleted,
	})
	if err != nil {
		return err
	}

	// update job state
	return updateJobState(ctx, db, shardID, "")
}

// StopShard a helper function to fail a shard, update the job state if all other shards are failed, and fail all executions.
func StopShard(ctx context.Context, db Store, shardID model.ShardID, reason string, userRequested bool) ([]model.ExecutionState, error) {
	// update shard state
	newShardState := model.ShardStateError
	unexpectedShardState := model.ShardStateCancelled
	if userRequested {
		newShardState = model.ShardStateCancelled
		unexpectedShardState = model.ShardStateError
	}
	err := db.UpdateShardState(ctx, UpdateShardStateRequest{
		ShardID: shardID,
		Condition: UpdateShardCondition{
			UnexpectedStates: []model.ShardStateType{
				model.ShardStateCompleted,
				unexpectedShardState,
			},
		},
		NewState: newShardState,
		Comment:  reason,
	})
	if err != nil {
		return nil, err
	}

	// update job state
	err = updateJobState(ctx, db, shardID, reason)
	if err != nil {
		return nil, err
	}

	// update execution state
	shardState, err := db.GetShardState(ctx, shardID)
	if err != nil {
		return nil, err
	}

	cancelledExecutions := make([]model.ExecutionState, 0)
	for _, execution := range shardState.Executions {
		if !execution.State.IsTerminal() {
			err = db.UpdateExecution(ctx, UpdateExecutionRequest{
				ExecutionID: execution.ID(),
				Condition: UpdateExecutionCondition{
					UnexpectedStates: []model.ExecutionStateType{
						model.ExecutionStateFailed,
						model.ExecutionStateCompleted,
					},
				},
				NewValues: model.ExecutionState{
					State:  model.ExecutionStateCanceled,
					Status: reason,
				},
			})
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msgf("failed to update execution state to Canceled. %s:%d",
					execution.JobID, execution.ShardIndex)
			} else {
				cancelledExecutions = append(cancelledExecutions, execution)
			}
		}
	}
	return cancelledExecutions, nil
}

func updateJobState(ctx context.Context, db Store, shardID model.ShardID, reason string) error {
	// update job state
	jobState, err := db.GetJobState(ctx, shardID.JobID)
	if err != nil {
		return err
	}
	cancelCount := 0
	errorCount := 0
	completedCount := 0
	totalCount := len(jobState.Shards)
	for _, shard := range jobState.Shards {
		if !shard.State.IsTerminal() {
			// if some shards are still running, don't update the job state
			return nil
		}
		if shard.State == model.ShardStateError {
			errorCount++
		} else if shard.State == model.ShardStateCompleted {
			completedCount++
		} else if shard.State == model.ShardStateCancelled {
			cancelCount++
		}
	}

	newJobState := model.JobStateCompleted
	if cancelCount > 0 {
		// if a single shard was canceled by the user, then the job is canceled
		newJobState = model.JobStateCancelled
	} else if errorCount >= totalCount {
		// if all shards failed, then the job failed
		newJobState = model.JobStateError
	} else if errorCount > 0 {
		// if some shards failed, then the job is partially completed
		newJobState = model.JobStatePartialError
	}

	if jobState.State != newJobState {
		err = db.UpdateJobState(ctx, UpdateJobStateRequest{
			JobID:     shardID.JobID,
			Condition: UpdateJobCondition{ExpectedState: jobState.State},
			NewState:  newJobState,
			Comment:   reason,
		})
		return err
	}
	return nil
}
