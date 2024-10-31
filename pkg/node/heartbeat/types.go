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

type Client interface {
	SendHeartbeat(ctx context.Context, sequence uint64) error
	Close(ctx context.Context) error
}
