package docker

import "github.com/bacalhau-project/bacalhau/pkg/models"

const DOCKER_EXECUTOR_COMPONENT = "Executor/Docker"

const (
	ExectorSpecValidationErr = "ExectorSpecValidationErr"
	ExecutionAlreadyStarted  = "ExecutionAlreadyStarted"
	ExecutionNotFound        = "ExecutionNotFound"
)

func NewDockerExecutorError(code models.ErrorCode, message string) *models.BaseError {
	return models.NewBaseError(message).
		WithCode(code).
		WithComponent(DOCKER_EXECUTOR_COMPONENT)
}
