package store

type ExecutionState int

const (
	ExecutionStateUndefined ExecutionState = iota
	ExecutionStateCreated
	ExecutionStateBidAccepted
	ExecutionStateRunning
	ExecutionStateWaitingVerification
	ExecutionStateResultAccepted
	ExecutionStatePublishing
	ExecutionStateCompleted
	ExecutionStateFailed
	ExecutionStateCancelled
)

// IsActive returns true if the execution is active
func (s ExecutionState) IsActive() bool {
	return s == ExecutionStateCreated || s == ExecutionStateBidAccepted || s == ExecutionStateRunning ||
		s == ExecutionStateWaitingVerification || s == ExecutionStateResultAccepted || s == ExecutionStatePublishing
}

// IsExecuting returns true if the execution is running in the backend
func (s ExecutionState) IsExecuting() bool {
	return s == ExecutionStateRunning || s == ExecutionStateWaitingVerification ||
		s == ExecutionStateResultAccepted || s == ExecutionStatePublishing
}

// IsTerminal returns true if the execution is terminal
func (s ExecutionState) IsTerminal() bool {
	return s == ExecutionStateCompleted || s == ExecutionStateFailed || s == ExecutionStateCancelled
}
