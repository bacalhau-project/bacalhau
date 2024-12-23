package bprotocol

import (
	"fmt"
)

const HeartbeatTopicFormat = "bacalhau.global.compute.%s.out.heartbeat"

// ComputeHeartbeatTopic returns the subject to publish heartbeat messages to.
// it publishes to the outgoing heartbeat subject of a specific compute node, which
// the orchestrator subscribes to.
func ComputeHeartbeatTopic(nodeID string) string {
	return fmt.Sprintf(HeartbeatTopicFormat, nodeID)
}

// OrchestratorHeartbeatSubscription returns the subject to subscribe for compute heartbeats.
// it subscribes for heartbeat messages from all compute nodes
func OrchestratorHeartbeatSubscription() string {
	return fmt.Sprintf(HeartbeatTopicFormat, "*")
}
