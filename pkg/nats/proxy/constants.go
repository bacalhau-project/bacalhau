package proxy

import (
	"fmt"
)

const (
	ComputeEndpointSubjectPrefix = "node.compute"
	CallbackSubjectPrefix        = "node.orchestrator"
	ManagementSubjectPrefix      = "node.management"

	AskForBid       = "AskForBid/v1"
	BidAccepted     = "BidAccepted/v1"
	BidRejected     = "BidRejected/v1"
	CancelExecution = "CancelExecution/v1"
	ExecutionLogs   = "ExecutionLogs/v1"

	OnBidComplete    = "OnBidComplete/v1"
	OnRunComplete    = "OnRunComplete/v1"
	OnCancelComplete = "OnCancelComplete/v1"
	OnComputeFailure = "OnComputeFailure/v1"

	RegisterNode    = "RegisterNode/v1"
	UpdateNodeInfo  = "UpdateNodeInfo/v1"
	UpdateResources = "UpdateResources/v1"
)

func computeEndpointPublishSubject(nodeID string, method string) string {
	return fmt.Sprintf("%s.%s.%s", ComputeEndpointSubjectPrefix, nodeID, method)
}

func computeEndpointSubscribeSubject(nodeID string) string {
	return fmt.Sprintf("%s.%s.>", ComputeEndpointSubjectPrefix, nodeID)
}

func callbackPublishSubject(method string) string {
	return fmt.Sprintf("%s.%s", CallbackSubjectPrefix, method)
}

func callbackSubscribeSubject() string {
	return fmt.Sprintf("%s.>", CallbackSubjectPrefix)
}

func managementPublishSubject(method string) string {
	return fmt.Sprintf("%s.%s", ManagementSubjectPrefix, method)
}

func managementSubscribeSubject() string {
	return fmt.Sprintf("%s.>", ManagementSubjectPrefix)
}
