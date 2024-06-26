package compute

import (
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// ErrExecTimeout is an error that is returned when an execution times out.
type ErrExecTimeout struct {
	ExecutionTimeout time.Duration
}

func NewErrExecTimeout(executionTimeout time.Duration) ErrExecTimeout {
	return ErrExecTimeout{
		ExecutionTimeout: executionTimeout,
	}
}

func (e ErrExecTimeout) Error() string {
	return fmt.Sprintf("Execution timed out after %s", e.ExecutionTimeout)
}

func (e ErrExecTimeout) Retryable() bool {
	return true
}

func (e ErrExecTimeout) Details() map[string]string {
	return map[string]string{
		models.DetailsKeyHint: "Try increasing the task timeout or reducing the task size",
	}
}
