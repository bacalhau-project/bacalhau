package bprotocol

const (
	ComputeServiceName    = "bacalhau.compute"
	AskForBidProtocolID   = "/bacalhau/compute/ask_for_bid/1.0.0"
	BidAcceptedProtocolID = "/bacalhau/compute/bid_accepted/1.0.0"
	BidRejectedProtocolID = "/bacalhau/compute/bid_rejected/1.0.0"
	CancelProtocolID      = "/bacalhau/compute/cancel/1.0.0"
	ExecutionLogsID       = "/bacalhau/compute/executionlogs/1.0.0"

	CallbackServiceName = "bacalhau.callback"
	OnBidComplete       = "/bacalhau/callback/on_bid_complete/1.0.0"
	OnRunComplete       = "/bacalhau/callback/on_run_complete/1.0.0"
	OnCancelComplete    = "/bacalhau/callback/on_cancel_complete/1.0.0"
	OnComputeFailure    = "/bacalhau/callback/on_compute_failure/1.0.0"
)
