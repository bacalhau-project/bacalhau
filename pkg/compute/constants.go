package compute

// Watcher event object types
const (
	EventObjectExecutionUpsert = "ExecutionUpsert"
	EventObjectExecutionEvent  = "ExecutionEvent"
)

const (
	AskForBidMessageType       = "AskForBid"
	BidAcceptedMessageType     = "BidAccepted"
	BidRejectedMessageType     = "BidRejected"
	CancelExecutionMessageType = "CancelExecution"

	BidResultMessageType    = "BidResult"
	RunResultMessageType    = "RunResult"
	ComputeErrorMessageType = "ComputeError"
)
