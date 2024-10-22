package noop

import (
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

const NoopExecutorComponent = "Executor/Noop"

func NewNoopExecutorError(code bacerrors.ErrorCode, message string) bacerrors.Error {
	return bacerrors.New("%s", message).
		WithCode(code).
		WithComponent(NoopExecutorComponent)
}
