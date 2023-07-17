//go:generate stringer -type=ExecutionState --trimprefix=ExecutionState --output state_string.go
package store

type ExecutionState int

const (
	ExecutionStateUndefined ExecutionState = iota
	ExecutionStateCreated
	ExecutionStateBidAccepted
	ExecutionStateRunning
	ExecutionStatePublishing
	ExecutionStateCompleted
	ExecutionStateFailed
	ExecutionStateCancelled
)

func ExecutionStateTypes() []ExecutionState {
	var res []ExecutionState
	for typ := ExecutionStateUndefined; typ <= ExecutionStateCancelled; typ++ {
		res = append(res, typ)
	}
	return res
}

// IsActive returns true if the execution is active
func (s ExecutionState) IsActive() bool {
	return s == ExecutionStateCreated || s == ExecutionStateBidAccepted || s == ExecutionStateRunning || s == ExecutionStatePublishing
}

// IsExecuting returns true if the execution is running in the backend
func (s ExecutionState) IsExecuting() bool {
	return s == ExecutionStateRunning || s == ExecutionStatePublishing
}

// IsTerminal returns true if the execution is terminal
func (s ExecutionState) IsTerminal() bool {
	return s == ExecutionStateCompleted || s == ExecutionStateFailed || s == ExecutionStateCancelled
}
