package store

func ValidateNewExecution(localExecutionState LocalExecutionState) error {
	// state must be either created, or bid accepted if the execution is pre-approved
	if localExecutionState.State != ExecutionStateCreated && localExecutionState.State != ExecutionStateBidAccepted {
		return NewErrInvalidExecutionState(
			localExecutionState.Execution.ID, localExecutionState.State, ExecutionStateCreated, ExecutionStateBidAccepted)
	}
	if localExecutionState.Revision != 1 {
		return NewErrInvalidExecutionRevision(localExecutionState.Execution.ID, localExecutionState.Revision, 1)
	}

	return nil
}
