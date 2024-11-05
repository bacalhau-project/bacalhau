package messages

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type CancelExecutionRequest struct {
	BaseRequest
	ExecutionID string
}

type RunResult struct {
	BaseResponse
	PublishResult    *models.SpecConfig
	RunCommandResult *models.RunCommandResult
}

type ComputeError struct {
	BaseResponse
}

func (e ComputeError) Error() string {
	return e.Message()
}
