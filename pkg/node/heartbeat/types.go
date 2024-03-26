package heartbeat

const heartbeatTopic = "heartbeat"

// Heartbeat represents a heartbeat message from a specific node.
// It contains the node ID and the sequence number of the heartbeat
// which is monotonically increasing (reboots aside). We do not
// use timestamps on the client, we rely solely on the server-side
// time to avoid clock drift issues.
type Heartbeat struct {
	NodeID   string
	Sequence uint64
}
