package node

import (
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
)

// CreateMessageSerDeRegistry creates a new payload registry.
func CreateMessageSerDeRegistry() (*ncl.MessageSerDeRegistry, error) {
	reg := ncl.NewMessageSerDeRegistry()
	err := errors.Join(
		reg.Register(heartbeat.HeartbeatMessageType, messages.Heartbeat{}),
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
