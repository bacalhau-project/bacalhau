package legacy

import "github.com/bacalhau-project/bacalhau/pkg/models"

type AskForBidRequest struct {
	RoutingMetadata
	// Execution specifies the job to be executed.
	Execution *models.Execution
	// WaitForApproval specifies whether the compute node should wait for the requester to approve the bid.
	// if set to true, the compute node will not start the execution until the requester approves the bid.
	// If set to false, the compute node will automatically start the execution after bidding and when resources are available.
	WaitForApproval bool
}

type AskForBidResponse struct {
	ExecutionMetadata
}

type BidAcceptedRequest struct {
	RoutingMetadata
	ExecutionID   string
	Accepted      bool
	Justification string
}

type BidAcceptedResponse struct {
	ExecutionMetadata
}

type BidRejectedRequest struct {
	RoutingMetadata
	ExecutionID   string
	Justification string
}

type BidRejectedResponse struct {
	ExecutionMetadata
}

// BidResult is the result of the compute node bidding on a job that is returned
// to the caller through a Callback.
type BidResult struct {
	RoutingMetadata
	ExecutionMetadata
	Accepted bool
	Wait     bool
	Event    models.Event
}
