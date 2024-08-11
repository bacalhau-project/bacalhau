package ncl

// NoopCheckpointer is a Checkpointer that does nothing
type NoopCheckpointer struct{}

// Checkpoint does nothing
func (n *NoopCheckpointer) Checkpoint(_ *Message) error { return nil }

// GetLastCheckpoint returns 0
func (n *NoopCheckpointer) GetLastCheckpoint() (int64, error) { return 0, nil }

// compile time check for interface conformance
var _ Checkpointer = &NoopCheckpointer{}
