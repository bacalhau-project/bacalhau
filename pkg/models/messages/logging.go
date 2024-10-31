package messages

type ExecutionLogsRequest struct {
	RoutingMetadata
	ExecutionID string
	Tail        bool
	Follow      bool
}

type ExecutionLogsResponse struct {
	Address           string
	ExecutionFinished bool
}
