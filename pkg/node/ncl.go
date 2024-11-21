package node

import (
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
)

// CreateMessageSerDeRegistry creates a new payload registry.
func CreateMessageSerDeRegistry() (*envelope.Registry, error) {
	reg := envelope.NewRegistry()
	err := errors.Join(
		reg.Register(heartbeat.HeartbeatMessageType, messages.Heartbeat{}),
		reg.Register(messages.AskForBidMessageType, messages.AskForBidRequest{}),
		reg.Register(messages.BidAcceptedMessageType, messages.BidAcceptedRequest{}),
		reg.Register(messages.BidRejectedMessageType, messages.BidRejectedRequest{}),
		reg.Register(messages.CancelExecutionMessageType, messages.CancelExecutionRequest{}),
		reg.Register(messages.BidResultMessageType, messages.BidResult{}),
		reg.Register(messages.RunResultMessageType, messages.RunResult{}),
		reg.Register(messages.ComputeErrorMessageType, messages.ComputeError{}),
	)
	return reg, err
}

const HeartbeatTopicFormat = "bacalhau.global.compute.%s.out.heartbeat"

// computeHeartbeatTopic returns the subject to publish heartbeat messages to.
// it publishes to the outgoing heartbeat subject of a specific compute node, which
// the orchestrator subscribes to.
func computeHeartbeatTopic(nodeID string) string {
	return fmt.Sprintf(HeartbeatTopicFormat, nodeID)
}

// orchestratorHeartbeatSubscription returns the subject to subscribe for compute heartbeats.
// it subscribes for heartbeat messages from all compute nodes
func orchestratorHeartbeatSubscription() string {
	return fmt.Sprintf(HeartbeatTopicFormat, "*")
}

// orchestratorSubjectSub returns the subject to subscribe to for orchestrator messages.
// it subscribes to outgoing messages from all compute nodes.
func orchestratorInSubscription() string {
	return "bacalhau.global.compute.*.out.msgs.>"
}

// orchestratorOutSubjectPrefix returns the subject to publish orchestrator messages to.
// it publishes to the incoming subject of a specific compute node.
func orchestratorOutSubjectPrefix(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.in.msgs", computeNodeID)
}

// computeInSubscription returns the subject to subscribe to for compute messages.
// it subscribes to incoming messages directed to its own node.
func computeInSubscription(nodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.in.msgs.>", nodeID)
}

// computeOutSubject returns the subject to publish compute messages to.
// it publishes to the outgoing subject of a specific compute node, which the
// orchestrator subscribes to.
func computeOutSubject(nodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.out.msgs", nodeID)
}
