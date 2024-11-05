package messages

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// BaseRequest is the base request model for all requests.
type BaseRequest struct {
	Events []*models.Event
}

// Message returns a request message if available.
func (r BaseRequest) Message() string {
	if len(r.Events) > 0 {
		return r.Events[0].Message
	}
	return ""
}

// BaseResponse is the base response model for all responses.
type BaseResponse struct {
	ExecutionID string
	JobID       string
	JobType     string
	Events      []*models.Event
}

func NewBaseResponse(execution *models.Execution) BaseResponse {
	if execution == nil {
		return BaseResponse{}
	}
	resp := BaseResponse{
		ExecutionID: execution.ID,
		Events:      []*models.Event{},
	}
	if execution.Job != nil {
		resp.JobID = execution.Job.ID
		resp.JobType = execution.Job.Type
	}
	return resp
}

// Message returns a response message if available.
func (r BaseResponse) Message() string {
	if len(r.Events) > 0 {
		return r.Events[0].Message
	}
	return ""
}
