package verifier

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// ErrInsufficientExecutions is returned when the number of executions is less than the minimum required
type ErrInsufficientExecutions struct {
	ShardID        string
	MinCount       int
	SubmittedCount int
}

func NewErrInsufficientExecutions(id string, minCount, submittedCount int) ErrInsufficientExecutions {
	return ErrInsufficientExecutions{ShardID: id, MinCount: minCount, SubmittedCount: submittedCount}
}

func (e ErrInsufficientExecutions) Error() string {
	return fmt.Sprintf("insufficient executions to verify shard %s: %d submitted, %d required", e.ShardID, e.SubmittedCount, e.MinCount)
}

// ErrMismatchingExecution is returned when the execution does not match the shard
type ErrMismatchingExecution struct {
	ShardID     model.ShardID
	ExecutionID model.ExecutionID
}

func NewErrMismatchingExecution(shardID model.ShardID, executionID model.ExecutionID) ErrMismatchingExecution {
	return ErrMismatchingExecution{ShardID: shardID, ExecutionID: executionID}
}

func (e ErrMismatchingExecution) Error() string {
	return fmt.Sprintf("execution %s does not match shard %s", e.ExecutionID, e.ShardID)
}

// ErrInvalidExecutionState is returned when the execution state is not valid for verification
type ErrInvalidExecutionState struct {
	ExecutionID model.ExecutionID
	State       model.ExecutionStateType
}

func NewErrInvalidExecutionState(id model.ExecutionID, state model.ExecutionStateType) ErrInvalidExecutionState {
	return ErrInvalidExecutionState{ExecutionID: id, State: state}
}

func (e ErrInvalidExecutionState) Error() string {
	return fmt.Sprintf("execution %s is in state %s", e.ExecutionID, e.State)
}

// ErrMissingVerificationProposal is returned when the verification proposal is missing
type ErrMissingVerificationProposal struct {
	ExecutionID model.ExecutionID
}

func NewErrMissingVerificationProposal(id model.ExecutionID) ErrMissingVerificationProposal {
	return ErrMissingVerificationProposal{ExecutionID: id}
}

func (e ErrMissingVerificationProposal) Error() string {
	return fmt.Sprintf("execution %s is missing verification proposal", e.ExecutionID)
}
