package store

import "fmt"

// ErrNilExecution is returned when the execution is nil
type ErrNilExecution struct{}

func NewErrNilExecution() ErrNilExecution {
	return ErrNilExecution{}
}

func (e ErrNilExecution) Error() string {
	return "execution is nil"
}

// ErrExecutionNotFound is returned when the execution is not found
type ErrExecutionNotFound struct {
	ExecutionID string
}

func NewErrExecutionNotFound(id string) ErrExecutionNotFound {
	return ErrExecutionNotFound{ExecutionID: id}
}

func (e ErrExecutionNotFound) Error() string {
	return "execution not found: " + e.ExecutionID
}

// ErrExecutionAlreadyExists is returned when an execution already exists
type ErrExecutionAlreadyExists struct {
	ExecutionID string
}

func NewErrExecutionAlreadyExists(id string) ErrExecutionAlreadyExists {
	return ErrExecutionAlreadyExists{ExecutionID: id}
}

func (e ErrExecutionAlreadyExists) Error() string {
	return "execution already exists: " + e.ExecutionID
}

// ErrInvalidExecutionState is returned when an execution is in an invalid state.
type ErrInvalidExecutionState struct {
	ExecutionID string
	Actual      ExecutionState
	Expected    ExecutionState
}

func NewErrInvalidExecutionState(id string, actual ExecutionState, expected ExecutionState) ErrInvalidExecutionState {
	return ErrInvalidExecutionState{ExecutionID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidExecutionState) Error() string {
	return "execution " + e.ExecutionID + " is in state " + e.Actual.String() + " but expected " + e.Expected.String()
}

// ErrInvalidExecutionVersion is returned when an execution has an invalid version.
type ErrInvalidExecutionVersion struct {
	ExecutionID string
	Actual      int
	Expected    int
}

func NewErrInvalidExecutionVersion(id string, actual int, expected int) ErrInvalidExecutionVersion {
	return ErrInvalidExecutionVersion{ExecutionID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidExecutionVersion) Error() string {
	return fmt.Sprintf("execution %s has version %d but expected %d", e.ExecutionID, e.Actual, e.Expected)
}
