package proxy

import "fmt"

const (
	ComputeEndpointSubjectPrefix = "node.compute"
	CallbackSubjectPrefix        = "node.orchestrator"

	AskForBid       = "AskForBid/1"
	BidAccepted     = "BidAccepted/1"
	BidRejected     = "BidRejected/1"
	CancelExecution = "CancelExecution/1"
	ExecutionLogs   = "ExecutionLogs/1"

	OnBidComplete    = "OnBidComplete/1"
	OnRunComplete    = "OnRunComplete/1"
	OnCancelComplete = "OnCancelComplete/1"
	OnComputeFailure = "OnComputeFailure/1"
)

func computeEndpointPublishSubject(nodeID string, method string) string {
	return fmt.Sprintf("%s.%s.%s", ComputeEndpointSubjectPrefix, nodeID, method)
}

func computeEndpointSubscribeSubject(nodeID string) string {
	return fmt.Sprintf("%s.%s.>", ComputeEndpointSubjectPrefix, nodeID)
}

func callbackPublishSubject(nodeID string, method string) string {
	return fmt.Sprintf("%s.%s.%s", CallbackSubjectPrefix, nodeID, method)
}

func callbackSubscribeSubject(nodeID string) string {
	return fmt.Sprintf("%s.%s.>", CallbackSubjectPrefix, nodeID)
}
