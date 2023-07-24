package jobstore

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
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
	Actual   model.JobStateType
	Expected model.JobStateType
}

func NewErrInvalidJobState(id string, actual model.JobStateType, expected model.JobStateType) ErrInvalidJobState {
	return ErrInvalidJobState{JobID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidJobState) Error() string {
	if e.Expected.IsUndefined() {
		return "job " + e.JobID + " is in unexpected state " + e.Actual.String() + "."
	}
	return "job " + e.JobID + " is in state " + e.Actual.String() + " but expected " + e.Expected.String()
}

// ErrInvalidJobVersion is returned when an job has an invalid version.
type ErrInvalidJobVersion struct {
	JobID    string
	Actual   int
	Expected int
}

func NewErrInvalidJobVersion(id string, actual int, expected int) ErrInvalidJobVersion {
	return ErrInvalidJobVersion{JobID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidJobVersion) Error() string {
	return fmt.Sprintf("job %s has version %d but expected %d", e.JobID, e.Actual, e.Expected)
}

// ErrJobAlreadyTerminal is returned when an job is already in terminal state and cannot be updated.
type ErrJobAlreadyTerminal struct {
	JobID    string
	Actual   model.JobStateType
	NewState model.JobStateType
}

func NewErrJobAlreadyTerminal(id string, actual model.JobStateType, newState model.JobStateType) ErrJobAlreadyTerminal {
	return ErrJobAlreadyTerminal{JobID: id, Actual: actual, NewState: newState}
}

func (e ErrJobAlreadyTerminal) Error() string {
	return fmt.Sprintf("job %s is in terminal state %s and cannot transition to %s",
		e.JobID, e.Actual.String(), e.NewState.String())
}

// ErrExecutionNotFound is returned when an job already exists
type ErrExecutionNotFound struct {
	ExecutionID model.ExecutionID
}

func NewErrExecutionNotFound(id model.ExecutionID) ErrExecutionNotFound {
	return ErrExecutionNotFound{ExecutionID: id}
}

func (e ErrExecutionNotFound) Error() string {
	return "execution not found: " + e.ExecutionID.String()
}

// ErrExecutionAlreadyExists is returned when an job already exists
type ErrExecutionAlreadyExists struct {
	ExecutionID model.ExecutionID
}

func NewErrExecutionAlreadyExists(id model.ExecutionID) ErrExecutionAlreadyExists {
	return ErrExecutionAlreadyExists{ExecutionID: id}
}

func (e ErrExecutionAlreadyExists) Error() string {
	return "execution already exists: " + e.ExecutionID.String()
}

// ErrInvalidExecutionState is returned when an execution is in an invalid state.
type ErrInvalidExecutionState struct {
	ExecutionID model.ExecutionID
	Actual      model.ExecutionStateType
	Expected    []model.ExecutionStateType
}

func NewErrInvalidExecutionState(
	id model.ExecutionID, actual model.ExecutionStateType, expected ...model.ExecutionStateType) ErrInvalidExecutionState {
	return ErrInvalidExecutionState{ExecutionID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidExecutionState) Error() string {
	if len(e.Expected) > 0 {
		return fmt.Sprintf("execution %s is in unexpted state %s", e.ExecutionID.String(), e.Actual.String())
	}
	return fmt.Sprintf("execution %s is in state %s", e.ExecutionID.String(), e.Actual.String())
}

// ErrInvalidExecutionVersion is returned when an execution has an invalid version.
type ErrInvalidExecutionVersion struct {
	ExecutionID model.ExecutionID
	Actual      int
	Expected    int
}

func NewErrInvalidExecutionVersion(id model.ExecutionID, actual int, expected int) ErrInvalidExecutionVersion {
	return ErrInvalidExecutionVersion{ExecutionID: id, Actual: actual, Expected: expected}
}

func (e ErrInvalidExecutionVersion) Error() string {
	return fmt.Sprintf("execution %s has version %d but expected %d", e.ExecutionID.String(), e.Actual, e.Expected)
}

// ErrExecutionAlreadyTerminal is returned when an execution is already in terminal state and cannot be updated.
type ErrExecutionAlreadyTerminal struct {
	ExecutionID model.ExecutionID
	Actual      model.ExecutionStateType
	NewState    model.ExecutionStateType
}

func NewErrExecutionAlreadyTerminal(
	id model.ExecutionID, actual model.ExecutionStateType, newState model.ExecutionStateType) ErrExecutionAlreadyTerminal {
	return ErrExecutionAlreadyTerminal{ExecutionID: id, Actual: actual, NewState: newState}
}

func (e ErrExecutionAlreadyTerminal) Error() string {
	return fmt.Sprintf("execution %s is in terminal state %s and cannot transition to %s",
		e.ExecutionID, e.Actual.String(), e.NewState.String())
}
