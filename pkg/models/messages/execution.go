package messages

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type CancelExecutionRequest struct {
	RoutingMetadata
	ExecutionID   string
	Justification string
}

type CancelExecutionResponse struct {
	ExecutionMetadata
}

type RunResult struct {
	RoutingMetadata
	ExecutionMetadata
	PublishResult    *models.SpecConfig
	RunCommandResult *models.RunCommandResult
}

type CancelResult struct {
	RoutingMetadata
	ExecutionMetadata
}

type ComputeError struct {
	RoutingMetadata
	ExecutionMetadata
	Event models.Event
}

func (e ComputeError) Error() string {
	return e.Event.Message
}
