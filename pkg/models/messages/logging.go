package messages

type ExecutionLogsRequest struct {
	ExecutionID string
	NodeID      string
	Tail        bool
	Follow      bool
}

type ExecutionLogsResponse struct {
	Address           string
	ExecutionFinished bool
}
