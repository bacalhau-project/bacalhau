package requests

// LogStreamRequest encapsulates the parameters required to retrieve a log stream.
type LogStreamRequest struct {
	RoutingMetadata
	ExecutionID string
	Tail        bool
	Follow      bool
}
