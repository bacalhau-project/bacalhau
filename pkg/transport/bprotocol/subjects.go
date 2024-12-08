package bprotocol

import (
	"fmt"
)

const HeartbeatTopicFormat = "bacalhau.global.compute.%s.out.heartbeat"

// computeHeartbeatTopic returns the subject to publish heartbeat messages to.
// it publishes to the outgoing heartbeat subject of a specific compute node, which
// the orchestrator subscribes to.
func ComputeHeartbeatTopic(nodeID string) string {
	return fmt.Sprintf(HeartbeatTopicFormat, nodeID)
}

// orchestratorHeartbeatSubscription returns the subject to subscribe for compute heartbeats.
// it subscribes for heartbeat messages from all compute nodes
func OrchestratorHeartbeatSubscription() string {
	return fmt.Sprintf(HeartbeatTopicFormat, "*")
}

// orchestratorSubjectSub returns the subject to subscribe to for orchestrator messages.
// it subscribes to outgoing messages from all compute nodes.
func OrchestratorInSubscription() string {
	return "bacalhau.global.compute.*.out.msgs"
}

// OrchestratorOutSubject returns the subject to publish orchestrator messages to.
// it publishes to the incoming subject of a specific compute node.
func orchestratorOutSubject(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.in.msgs", computeNodeID)
}

// computeInSubscription returns the subject to subscribe to for compute messages.
// it subscribes to incoming messages directed to its own node.
func ComputeInSubscription(nodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.in.msgs", nodeID)
}

// computeOutSubject returns the subject to publish compute messages to.
// it publishes to the outgoing subject of a specific compute node, which the
// orchestrator subscribes to.
func ComputeOutSubject(nodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.out.msgs", nodeID)
}
