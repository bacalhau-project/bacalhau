package jobstore

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const JobStoreComponent = "JobStore"
const (
	ConflictJobState         bacerrors.ErrorCode = "ConflictJobState"
	MultipleJobsFound        bacerrors.ErrorCode = "MultipleJobsFound"
	MultipleExecutionsFound  bacerrors.ErrorCode = "MultipleExecutionsFound"
	MultipleEvaluationsFound bacerrors.ErrorCode = "MultipleEvaluationsFound"
	ConflictJobVersion       bacerrors.ErrorCode = "ConflictJobVersion"
)

func NewErrJobNotFound(id string) bacerrors.Error {
	return bacerrors.New("job not found: %s", id).
		WithCode(bacerrors.NotFoundError).
		WithComponent(JobStoreComponent)
}

func NewErrMultipleJobsFound(id string) bacerrors.Error {
	return bacerrors.New("multiple jobs found for id %s", id).
		WithCode(MultipleJobsFound).
		WithComponent(JobStoreComponent).
		WithHint("Use full job ID")
}

func NewErrJobAlreadyExists(id string) bacerrors.Error {
	return bacerrors.New("job already exists: %s", id).
		WithCode(bacerrors.ResourceInUse).
		WithComponent(JobStoreComponent)
}

func NewErrInvalidJobState(id string, actual models.JobStateType, expected models.JobStateType) bacerrors.Error {
	var errorFormat string
	if expected.IsUndefined() {
		errorFormat = "job %s is in unexpected state %s"
	} else {
		errorFormat = "job %s is in state %s but expected %s"
	}

	return bacerrors.New(errorFormat, id, actual).
		WithCode(ConflictJobState).
		WithComponent(JobStoreComponent)
}

func NewErrInvalidJobVersion(id string, actual, expected uint64) bacerrors.Error {
	errorMessage := fmt.Sprintf("job %s has version %d but expected %d", id, actual, expected)
	return bacerrors.New("%s", errorMessage).
		WithCode(ConflictJobVersion).
		WithComponent(JobStoreComponent)
}

func NewErrJobAlreadyTerminal(id string, actual models.JobStateType, newState models.JobStateType) bacerrors.Error {
	errorMessage := fmt.Sprintf("job %s is in terminal state %s and cannot transition to %s", id, actual, newState)
	return bacerrors.New("%s", errorMessage).
		WithCode(ConflictJobState).
		WithComponent(JobStoreComponent)
}

func NewErrExecutionNotFound(id string) bacerrors.Error {
	return bacerrors.New("execution not found: %s", id).
		WithCode(bacerrors.NotFoundError).
		WithComponent(JobStoreComponent)
}

func NewErrMultipleExecutionsFound(id string) bacerrors.Error {
	return bacerrors.New("multiple executions found for id %s", id).
		WithCode(MultipleExecutionsFound).
		WithComponent(JobStoreComponent).
		WithHint("Use full execution ID")
}

func NewErrExecutionAlreadyExists(id string) bacerrors.Error {
	return bacerrors.New("execution already exists %s", id).
		WithCode(bacerrors.ResourceInUse).
		WithComponent(JobStoreComponent)
}

func NewErrInvalidExecutionState(id string, actual models.ExecutionStateType, expected ...models.ExecutionStateType) bacerrors.Error {
	var errorMessage string
	if len(expected) > 0 {
		errorMessage = fmt.Sprintf("execution %s is in unexpected state %s", id, actual)
	} else {
		errorMessage = fmt.Sprintf("execution %s is in state %s, but expected %s", id, actual, expected)
	}
	return bacerrors.New("%s", errorMessage).
		WithCode(ConflictJobState).
		WithComponent(JobStoreComponent)
}

func NewErrInvalidExecutionDesiredState(
	id string, actual models.ExecutionDesiredStateType, expected ...models.ExecutionDesiredStateType) bacerrors.Error {
	var errorMessage string
	if len(expected) > 0 {
		errorMessage = fmt.Sprintf("execution %s is in unexpected state %s", id, actual)
	} else {
		errorMessage = fmt.Sprintf("execution %s is in state %s, but expected %s", id, actual, expected)
	}
	return bacerrors.New("%s", errorMessage).
		WithCode(ConflictJobState).
		WithComponent(JobStoreComponent)
}

func NewErrInvalidExecutionVersion(id string, actual, expected uint64) bacerrors.Error {
	return bacerrors.New("execution %s has version %d but expected %d", id, actual, expected).
		WithCode(ConflictJobVersion).
		WithComponent(JobStoreComponent)
}

func NewErrExecutionAlreadyTerminal(id string, actual models.ExecutionStateType, newState models.ExecutionStateType) bacerrors.Error {
	return bacerrors.New("execution %s is in terminal state %s and cannot transition to %s", id, actual, newState).
		WithCode(ConflictJobState).
		WithComponent(JobStoreComponent)
}

func NewErrEvaluationAlreadyExists(id string) bacerrors.Error {
	return bacerrors.New("evaluation already exists: %s", id).
		WithCode(bacerrors.ResourceInUse).
		WithComponent(JobStoreComponent)
}

func NewErrEvaluationNotFound(id string) bacerrors.Error {
	return bacerrors.New("evaluation not found: %s", id).
		WithCode(bacerrors.NotFoundError).
		WithComponent(JobStoreComponent)
}

func NewErrMultipleEvaluationsFound(id string) bacerrors.Error {
	return bacerrors.New("multiple evaluations found for id %s", id).
		WithCode(MultipleEvaluationsFound).
		WithComponent(JobStoreComponent).
		WithHint("Use full evaluation ID")
}

func NewJobStoreError(message string) bacerrors.Error {
	return bacerrors.New("%s", message).
		WithCode(bacerrors.BadRequestError).
		WithComponent(JobStoreComponent)
}

func NewBadRequestError(message string) bacerrors.Error {
	return bacerrors.New("%s", message).
		WithCode(bacerrors.BadRequestError).
		WithComponent(JobStoreComponent)
}
