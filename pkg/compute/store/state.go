//go:generate stringer -type=ExecutionState --trimprefix=ExecutionState --output state_string.go
package store

type LocalStateType int

const (
	ExecutionStateUndefined LocalStateType = iota
	ExecutionStateCreated
	ExecutionStateBidAccepted
	ExecutionStateRunning
	ExecutionStatePublishing
	ExecutionStateCompleted
	ExecutionStateFailed
	ExecutionStateCancelled
)

func ExecutionStateTypes() []LocalStateType {
	var res []LocalStateType
	for typ := ExecutionStateUndefined; typ <= ExecutionStateCancelled; typ++ {
		res = append(res, typ)
	}
	return res
}

// IsUndefined returns true if the execution state is undefined
func (s LocalStateType) IsUndefined() bool {
	return s == ExecutionStateUndefined
}

// IsActive returns true if the execution is active
func (s LocalStateType) IsActive() bool {
	return s == ExecutionStateCreated || s == ExecutionStateBidAccepted || s == ExecutionStateRunning || s == ExecutionStatePublishing
}

// IsExecuting returns true if the execution is running in the backend
func (s LocalStateType) IsExecuting() bool {
	return s == ExecutionStateRunning || s == ExecutionStatePublishing
}

// IsTerminal returns true if the execution is terminal
func (s LocalStateType) IsTerminal() bool {
	return s == ExecutionStateCompleted || s == ExecutionStateFailed || s == ExecutionStateCancelled
}
