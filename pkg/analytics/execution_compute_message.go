package analytics

const ComputeMessageExecutionEventType = "bacalhau.execution_v1.compute_message"

type ExecutionComputeMessage struct {
	JobID          string `json:"job_id,omitempty"`
	ExecutionID    string `json:"execution_id,omitempty"`
	ComputeMessage string `json:"compute_message,omitempty"`
}

func NewComputeMessageExecutionEvent(jobID string, executionID string, computeMessage string) *Event {
	return NewEvent(ComputeMessageExecutionEventType, ExecutionComputeMessage{
		JobID:          jobID,
		ExecutionID:    executionID,
		ComputeMessage: computeMessage,
	})
}
