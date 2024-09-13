package noop

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const NoopExecutorComponent = "Executor/Noop"

func NewNoopExecutorError(code models.ErrorCode, message string) *models.BaseError {
	return models.NewBaseError(message).
		WithCode(code).
		WithComponent(NoopExecutorComponent)
}
