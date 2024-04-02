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

// ErrExecutionsNotFoundForJob is returned when the execution is not found for a given job
type ErrExecutionsNotFoundForJob struct {
	JobID string
}

func NewErrExecutionsNotFoundForJob(id string) ErrExecutionsNotFoundForJob {
	return ErrExecutionsNotFoundForJob{JobID: id}
}

func (e ErrExecutionsNotFoundForJob) Error() string {
	return "no executions found for job: " + e.JobID
}

// ErrJobNotFound is returned when the a job isn't found when trying to get its executions
type ErrJobNotFound struct {
	JobID string
}

func NewErrJobNotFound(id string) ErrJobNotFound {
	return ErrJobNotFound{JobID: id}
}

func (e ErrJobNotFound) Error() string {
	return "job not found: " + e.JobID
}

// ErrExecutionHistoryNotFound is returned when the execution is not found
type ErrExecutionHistoryNotFound struct {
	ExecutionID string
}

func NewErrExecutionHistoryNotFound(id string) ErrExecutionHistoryNotFound {
	return ErrExecutionHistoryNotFound{ExecutionID: id}
}

func (e ErrExecutionHistoryNotFound) Error() string {
	return "no history found for execution: " + e.ExecutionID
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
	Actual      LocalExecutionStateType
	Expected    []LocalExecutionStateType
}

func NewErrInvalidExecutionState(id string, actual LocalExecutionStateType, expected ...LocalExecutionStateType) ErrInvalidExecutionState {
	return ErrInvalidExecutionState{ExecutionID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidExecutionState) Error() string {
	return fmt.Sprintf("execution %s is in state %s but expected one of %v", e.ExecutionID, e.Actual, e.Expected)
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

// ErrExecutionAlreadyTerminal is returned when an execution is already in terminal state and cannot be updated.
type ErrExecutionAlreadyTerminal struct {
	ExecutionID string
	Actual      LocalExecutionStateType
	NewState    LocalExecutionStateType
}

func NewErrExecutionAlreadyTerminal(
	id string, actual LocalExecutionStateType, newState LocalExecutionStateType) ErrExecutionAlreadyTerminal {
	return ErrExecutionAlreadyTerminal{ExecutionID: id, Actual: actual, NewState: newState}
}

func (e ErrExecutionAlreadyTerminal) Error() string {
	return fmt.Sprintf("execution %s is in terminal state %s and cannot transition to %s",
		e.ExecutionID, e.Actual.String(), e.NewState.String())
}
