package messages

import "github.com/bacalhau-project/bacalhau/pkg/models"

type AskForBidRequest struct {
	BaseRequest
	// Execution specifies the job to be executed.
	Execution *models.Execution
}

type BidAcceptedRequest struct {
	BaseRequest
	ExecutionID string
	Accepted    bool
}

type BidRejectedRequest struct {
	BaseRequest
	ExecutionID string
}

// BidResult is the result of the compute node bidding on a job that is returned
// to the caller through a Callback.
type BidResult struct {
	BaseResponse
	Accepted bool
}
