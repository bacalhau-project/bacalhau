package messages

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
	return ExecutionMetadata{
		ExecutionID: execution.ID,
		JobID:       execution.Job.ID,
	}
}
