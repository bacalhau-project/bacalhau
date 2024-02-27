package proxy

import "fmt"

const (
	ComputeEndpointSubjectPrefix = "node.compute"
	CallbackSubjectPrefix        = "node.orchestrator"
	RegistrationSubjectPrefix    = "node.registration"

	AskForBid       = "AskForBid/v1"
	BidAccepted     = "BidAccepted/v1"
	BidRejected     = "BidRejected/v1"
	CancelExecution = "CancelExecution/v1"
	ExecutionLogs   = "ExecutionLogs/v1"

	OnBidComplete    = "OnBidComplete/v1"
	OnRunComplete    = "OnRunComplete/v1"
	OnCancelComplete = "OnCancelComplete/v1"
	OnComputeFailure = "OnComputeFailure/v1"

	Register = "Register/v1"
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

func registrationPublishSubject(method string) string {
	return fmt.Sprintf("%s.%s", RegistrationSubjectPrefix, method)
}

func registrationSubscribeSubject() string {
	return fmt.Sprintf("%s.>", RegistrationSubjectPrefix)
}
