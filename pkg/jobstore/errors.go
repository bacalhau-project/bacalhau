package jobstore

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// ErrJobNotFound is returned when the job is not found
type ErrJobNotFound struct {
	JobID string
}

func NewErrJobNotFound(id string) ErrJobNotFound {
	return ErrJobNotFound{JobID: id}
}

func (e ErrJobNotFound) Error() string {
	return "job not found: " + e.JobID
}

// ErrJobAlreadyExists is returned when an job already exists
type ErrJobAlreadyExists struct {
	JobID string
}

func NewErrJobAlreadyExists(id string) ErrJobAlreadyExists {
	return ErrJobAlreadyExists{JobID: id}
}

func (e ErrJobAlreadyExists) Error() string {
	return "job already exists: " + e.JobID
}

// ErrInvalidJobState is returned when an job is in an invalid state.
type ErrInvalidJobState struct {
	JobID    string
	Actual   models.JobStateType
	Expected models.JobStateType
}

func NewErrInvalidJobState(id string, actual models.JobStateType, expected models.JobStateType) ErrInvalidJobState {
	return ErrInvalidJobState{JobID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidJobState) Error() string {
	if e.Expected.IsUndefined() {
		return fmt.Sprintf("job %s is in unexpected state %s", e.JobID, e.Actual)
	}
	return fmt.Sprintf("job %s is in state %s but expected %s", e.JobID, e.Actual, e.Expected)
}

// ErrInvalidJobVersion is returned when an job has an invalid version.
type ErrInvalidJobVersion struct {
	JobID    string
	Actual   uint64
	Expected uint64
}

func NewErrInvalidJobVersion(id string, actual, expected uint64) ErrInvalidJobVersion {
	return ErrInvalidJobVersion{JobID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidJobVersion) Error() string {
	return fmt.Sprintf("job %s has version %d but expected %d", e.JobID, e.Actual, e.Expected)
}

// ErrJobAlreadyTerminal is returned when an job is already in terminal state and cannot be updated.
type ErrJobAlreadyTerminal struct {
	JobID    string
	Actual   models.JobStateType
	NewState models.JobStateType
}

func NewErrJobAlreadyTerminal(id string, actual models.JobStateType, newState models.JobStateType) ErrJobAlreadyTerminal {
	return ErrJobAlreadyTerminal{JobID: id, Actual: actual, NewState: newState}
}

func (e ErrJobAlreadyTerminal) Error() string {
	return fmt.Sprintf("job %s is in terminal state %s and cannot transition to %s",
		e.JobID, e.Actual, e.NewState)
}

// ErrExecutionNotFound is returned when an job already exists
type ErrExecutionNotFound struct {
	ExecutionID string
}

func NewErrExecutionNotFound(id string) ErrExecutionNotFound {
	return ErrExecutionNotFound{ExecutionID: id}
}

func (e ErrExecutionNotFound) Error() string {
	return "execution not found: " + e.ExecutionID
}

// ErrExecutionAlreadyExists is returned when an job already exists
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
	Actual      models.ExecutionStateType
	Expected    []models.ExecutionStateType
}

func NewErrInvalidExecutionState(
	id string, actual models.ExecutionStateType, expected ...models.ExecutionStateType) ErrInvalidExecutionState {
	return ErrInvalidExecutionState{ExecutionID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidExecutionState) Error() string {
	if len(e.Expected) > 0 {
		return fmt.Sprintf("execution %s is in unexpted state %s", e.ExecutionID, e.Actual)
	}
	return fmt.Sprintf("execution %s is in state %s, but expeted %s", e.ExecutionID, e.Actual, e.Expected)
}

// ErrInvalidExecutionVersion is returned when an execution has an invalid version.
type ErrInvalidExecutionVersion struct {
	ExecutionID string
	Actual      uint64
	Expected    uint64
}

func NewErrInvalidExecutionVersion(id string, actual, expected uint64) ErrInvalidExecutionVersion {
	return ErrInvalidExecutionVersion{ExecutionID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidExecutionVersion) Error() string {
	return fmt.Sprintf("execution %s has version %d but expected %d", e.ExecutionID, e.Actual, e.Expected)
}

// ErrExecutionAlreadyTerminal is returned when an execution is already in terminal state and cannot be updated.
type ErrExecutionAlreadyTerminal struct {
	ExecutionID string
	Actual      models.ExecutionStateType
	NewState    models.ExecutionStateType
}

func NewErrExecutionAlreadyTerminal(
	id string, actual models.ExecutionStateType, newState models.ExecutionStateType) ErrExecutionAlreadyTerminal {
	return ErrExecutionAlreadyTerminal{ExecutionID: id, Actual: actual, NewState: newState}
}

func (e ErrExecutionAlreadyTerminal) Error() string {
	return fmt.Sprintf("execution %s is in terminal state %s and cannot transition to %s",
		e.ExecutionID, e.Actual, e.NewState)
}
