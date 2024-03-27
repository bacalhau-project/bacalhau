package heartbeat

import "time"

const (
	heartbeatTopic               = "heartbeat"
	heartbeatQueueCheckFrequency = 5 * time.Second

	unhealthyAfter = 30 * time.Second
	unknownAfter   = 60 * time.Second
)

// Heartbeat represents a heartbeat message from a specific node.
// It contains the node ID and the sequence number of the heartbeat
// which is monotonically increasing (reboots aside). We do not
// use timestamps on the client, we rely solely on the server-side
// time to avoid clock drift issues.
type Heartbeat struct {
	NodeID   string
	Sequence uint64
}
