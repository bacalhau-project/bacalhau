package heartbeat

import (
	"context"
)

const (
	// HeartbeatMessageType is the message type for heartbeats
	HeartbeatMessageType = "heartbeat"
)

//	legacyHeartbeatTopic is the topic where the heartbeat messages are sent prior to v1.5.
//
// TODO: Remove this legacy heartbeat topic with v1.6
const legacyHeartbeatTopic = "heartbeat"

// Heartbeat represents a heartbeat message from a specific node.
// It contains the node ID and the sequence number of the heartbeat
// which is monotonically increasing (reboots aside). We do not
// use timestamps on the client, we rely solely on the server-side
// time to avoid clock drift issues.
type Heartbeat struct {
	NodeID   string
	Sequence uint64
}

type Client interface {
	SendHeartbeat(ctx context.Context, sequence uint64) error
	Close(ctx context.Context) error
}
