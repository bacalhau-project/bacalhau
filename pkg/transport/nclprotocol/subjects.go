package nclprotocol

import (
	"fmt"
)

func NatsSubjectOrchestratorInCtrl() string {
	return "bacalhau.global.compute.*.out.ctrl"
}

func NatsSubjectOrchestratorInRequests() string {
	return "bacalhau.global.compute.*.out.requests"
}

func NatsSubjectOrchestratorInMsgs(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.out.msgs", computeNodeID)
}

func NatsSubjectOrchestratorOutMsgs(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.in.msgs", computeNodeID)
}

func NatsSubjectComputeInMsgs(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.in.msgs", computeNodeID)
}

func NatsSubjectComputeOutCtrl(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.out.ctrl", computeNodeID)
}

func NatsSubjectComputeOutRequests(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.out.requests", computeNodeID)
}

func NatsSubjectComputeOutMsgs(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.out.msgs", computeNodeID)
}
