package legacy

import "github.com/bacalhau-project/bacalhau/pkg/models"

type RoutingMetadata struct {
	SourcePeerID string
	TargetPeerID string
}

type ExecutionMetadata struct {
	ExecutionID string
	JobID       string
}

func NewExecutionMetadata(execution *models.Execution) ExecutionMetadata {
	if execution == nil {
		return ExecutionMetadata{}
	}
	if execution.Job == nil {
		return ExecutionMetadata{ExecutionID: execution.ID}
	}
	return ExecutionMetadata{
		ExecutionID: execution.ID,
		JobID:       execution.Job.ID,
	}
}
