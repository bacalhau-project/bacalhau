package messages

const (
	AskForBidMessageType       = "AskForBid"
	BidAcceptedMessageType     = "BidAccepted"
	BidRejectedMessageType     = "BidRejected"
	CancelExecutionMessageType = "CancelExecution"

	BidResultMessageType    = "BidResult"
	RunResultMessageType    = "RunResult"
	ComputeErrorMessageType = "ComputeError"

	HandshakeRequestMessageType      = "transport.HandshakeRequest"
	HeartbeatRequestMessageType      = "transport.HeartbeatRequest"
	NodeInfoUpdateRequestMessageType = "transport.UpdateNodeInfoRequest"
	ShutdownNoticeRequestMessageType = "transport.ShutdownNoticeRequest"

	HandshakeResponseType      = "transport.HandshakeResponse"
	HeartbeatResponseType      = "transport.HeartbeatResponse"
	NodeInfoUpdateResponseType = "transport.UpdateNodeInfoResponse"
	ShutdownNoticeResponseType = "transport.ShutdownNoticeResponse"
)
