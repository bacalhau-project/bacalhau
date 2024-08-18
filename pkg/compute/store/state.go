//go:generate stringer -type=LocalExecutionStateType --trimprefix=ExecutionState --output state_string.go
package store

import (
	"strings"
)

type LocalExecutionStateType int

const (
	ExecutionStateUndefined LocalExecutionStateType = iota
	ExecutionStateCreated
	ExecutionStateBidAccepted
	ExecutionStateRunning
	ExecutionStatePublishing
	ExecutionStateCompleted
	ExecutionStateFailed
	ExecutionStateCancelled
)

func ExecutionStateTypes() []LocalExecutionStateType {
	var res []LocalExecutionStateType
	for typ := ExecutionStateUndefined; typ <= ExecutionStateCancelled; typ++ {
		res = append(res, typ)
	}
	return res
}

// IsUndefined returns true if the execution state is undefined
func (s LocalExecutionStateType) IsUndefined() bool {
	return s == ExecutionStateUndefined
}

// IsActive returns true if the execution is active
func (s LocalExecutionStateType) IsActive() bool {
	return s == ExecutionStateCreated || s == ExecutionStateBidAccepted || s == ExecutionStateRunning || s == ExecutionStatePublishing
}

// IsExecuting returns true if the execution is running in the backend
func (s LocalExecutionStateType) IsExecuting() bool {
	return s == ExecutionStateRunning || s == ExecutionStatePublishing
}

// IsTerminal returns true if the execution is terminal
func (s LocalExecutionStateType) IsTerminal() bool {
	return s == ExecutionStateCompleted || s == ExecutionStateFailed || s == ExecutionStateCancelled
}

func (s LocalExecutionStateType) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *LocalExecutionStateType) UnmarshalText(text []byte) (err error) {
	name := strings.TrimSpace(string(text))
	for _, typ := range ExecutionStateTypes() {
		if strings.EqualFold(typ.String(), name) {
			*s = typ
			return
		}
	}
	return
}
