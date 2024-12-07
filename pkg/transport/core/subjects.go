package core

import (
	"fmt"
)

const (
	NatsSubjectSuffixCtrl = "ctrl"

	NatsSubjectSuffixMsgs = "msgs"
)

func NatsSubjectOrchestratorInCtrl() string {
	return "bacalhau.global.compute.*.in.ctrl"
}

func NatsSubjectOrchestratorInMsgs(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.out.msgs", computeNodeID)
}

func NatsSubjectOrchestratorOutMsgs(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.in.msgs", computeNodeID)
}

func NatsSubjectComputeBase(nodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s", nodeID)
}

func NatsSubjectComputeInCtrl(nodeID string) string {
	return fmt.Sprintf("%s.in.%s", NatsSubjectComputeBase(nodeID), NatsSubjectSuffixCtrl)
}

func NatsSubjectComputeInMsgs(nodeID string) string {
	return fmt.Sprintf("%s.in.%s", NatsSubjectComputeBase(nodeID), NatsSubjectSuffixMsgs)
}

func NatsSubjectComputeOutCtrl(nodeID string) string {
	return fmt.Sprintf("%s.out.%s", NatsSubjectComputeBase(nodeID), NatsSubjectSuffixCtrl)
}

func NatsSubjectComputeOutMsgs(nodeID string) string {
	return fmt.Sprintf("%s.out.%s", NatsSubjectComputeBase(nodeID), NatsSubjectSuffixMsgs)
}
