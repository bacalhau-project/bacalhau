package node

import (
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
)

// CreateMessageSerDeRegistry creates a new payload registry.
func CreateMessageSerDeRegistry() (*ncl.MessageSerDeRegistry, error) {
	reg := ncl.NewMessageSerDeRegistry()
	err := errors.Join(
		reg.Register(heartbeat.HeartbeatMessageType, heartbeat.Heartbeat{}),
		reg.Register(compute.AskForBidMessageType, compute.AskForBidRequest{}),
		reg.Register(compute.BidAcceptedMessageType, compute.BidAcceptedRequest{}),
		reg.Register(compute.BidRejectedMessageType, compute.BidRejectedRequest{}),
		reg.Register(compute.CancelExecutionMessageType, compute.CancelExecutionRequest{}),
		reg.Register(compute.BidResultMessageType, compute.BidResult{}),
		reg.Register(compute.RunResultMessageType, compute.RunResult{}),
		reg.Register(compute.ComputeErrorMessageType, compute.ComputeError{}),
	)
	return reg, err
}

// orchestratorSubjectSub returns the subject to subscribe to for orchestrator messages.
// it subscribes to outgoing messages from all compute nodes.
func orchestratorInSubscription() string {
	return "bacalhau.global.compute.*.out.>"
}

// orchestratorOutSubjectPrefix returns the subject to publish orchestrator messages to.
// it publishes to the incoming subject of a specific compute node.
func orchestratorOutSubjectPrefix(computeNodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.in", computeNodeID)
}

// computeInSubscription returns the subject to subscribe to for compute messages.
// it subscribes to incoming messages directed to its own node.
func computeInSubscription(nodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.in.*", nodeID)
}

// computeOutSubject returns the subject to publish compute messages to.
// it publishes to the outgoing subject of a specific compute node, which the
// orchestrator subscribes to.
func computeOutSubject(nodeID string) string {
	return fmt.Sprintf("bacalhau.global.compute.%s.out", nodeID)
}
