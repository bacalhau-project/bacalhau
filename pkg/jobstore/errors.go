package jobstore

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	JOB_STORE_COMPONENT = "JBS"
)

func NewErrJobNotFound(id string) *models.BaseError {
	return models.NewBaseError("job not found: %s", id).WithCode(models.NewErrorCode(JOB_STORE_COMPONENT, 404))
}

func NewErrMultipleJobsFound(id string) *models.BaseError {
	return models.NewBaseError("multiple jobs found for id %s", id).WithCode(models.NewErrorCode(JOB_STORE_COMPONENT, 400))
}

func NewErrJobAlreadyExists(id string) *models.BaseError {
	return models.NewBaseError("job already exists: %s", id)
}

func NewErrInvalidJobState(id string, actual models.JobStateType, expected models.JobStateType) *models.BaseError {
	var errorFormat string
	if expected.IsUndefined() {
		errorFormat = "job %s is in unexpected state %s"
	} else {
		errorFormat = "job %s is in state %s but expected %s"
	}

	return models.NewBaseError(errorFormat, id, actual)
}

func NewErrInvalidJobVersion(id string, actual, expected uint64) *models.BaseError {
	errorMessage := fmt.Sprintf("job %s has version %d but expected %d", id, actual, expected)
	return models.NewBaseError(errorMessage)
}

func NewErrJobAlreadyTerminal(id string, actual models.JobStateType, newState models.JobStateType) *models.BaseError {
	errorMessage := fmt.Sprintf("job %s is in terminal state %s and connt transition to %s", id, actual, newState)
	return models.NewBaseError(errorMessage)
}

func NewErrExecutionNotFound(id string) *models.BaseError {
	return models.NewBaseError("execution not found: %s", id)
}

func NewErrExecutionAlreadyExists(id string) *models.BaseError {
	return models.NewBaseError("execution already exists %s", id)
}

func NewErrInvalidExecutionState(id string, actual models.ExecutionStateType, expected ...models.ExecutionStateType) *models.BaseError {
	var errorMessage string
	if len(expected) > 0 {
		errorMessage = fmt.Sprintf("execution %s is in unexpected state %s", id, actual)
	} else {
		errorMessage = fmt.Sprintf("execution %s is in state %s, but expected %s", id, actual, expected)
	}
	return models.NewBaseError(errorMessage)
}

func NewErrInvalidExecutionVersion(id string, actual, expected uint64) *models.BaseError {
	return models.NewBaseError("execution %s has version %d but expected %d", id, actual, expected)

}

func NewErrExecutionAlreadyTerminal(id string, actual models.ExecutionStateType, newState models.ExecutionStateType) *models.BaseError {
	return models.NewBaseError("execution %s is in terminal state %s and cannot transition to %s", id, actual, newState)
}

func NewJobStoreError(message string) *models.BaseError {
	return models.NewBaseError(message)
}
