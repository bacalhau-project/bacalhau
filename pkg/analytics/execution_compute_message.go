package analytics

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const ComputeMessageExecutionEventType = "bacalhau.execution_v1.compute_message"

type ExecutionComputeMessage struct {
	JobID            string `json:"job_id,omitempty"`
	ExecutionID      string `json:"execution_id,omitempty"`
	ComputeMessage   string `json:"compute_message,omitempty"`
	ComputeErrorCode string `json:"compute_state_error_code,omitempty"`
}

func NewComputeMessageExecutionEvent(e models.Execution) *Event {
	var errorCode string
	if e.ComputeState.Details != nil {
		errorCode = e.ComputeState.Details[models.DetailsKeyErrorCode]
	}
	return NewEvent(ComputeMessageExecutionEventType, ExecutionComputeMessage{
		JobID:            e.JobID,
		ExecutionID:      e.ID,
		ComputeMessage:   e.ComputeState.Message,
		ComputeErrorCode: errorCode,
	})
}
