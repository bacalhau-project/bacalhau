package proxy

import (
	"fmt"
)

const (
	ComputeEndpointSubjectPrefix = "node.compute"
	CallbackSubjectPrefix        = "node.orchestrator"
	ManagementSubjectPrefix      = "node.management"

	ExecutionLogs = "ExecutionLogs/v1"

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

func callbackPublishSubject(nodeID string, method string) string {
	return fmt.Sprintf("%s.%s.%s", CallbackSubjectPrefix, nodeID, method)
}

func managementPublishSubject(nodeID string, method string) string {
	return fmt.Sprintf("%s.%s.%s", ManagementSubjectPrefix, nodeID, method)
}

func managementSubscribeSubject() string {
	return fmt.Sprintf("%s.>", ManagementSubjectPrefix)
}
