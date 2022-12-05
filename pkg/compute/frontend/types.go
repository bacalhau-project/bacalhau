package frontend

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// Service is the frontend and entry point to the compute node. Requesters, whether through API, CLI or other means, do
// interact with the frontend service to submit jobs, ask for bids, accept or reject bids, etc.
type Service interface {
	// AskForBid asks for a bid for a given job and shard IDs, which will assign executionIDs for each shard the node
	// is interested in bidding on.
	AskForBid(context.Context, AskForBidRequest) (AskForBidResponse, error)
	// BidAccepted accepts a bid for a given executionID, which will trigger executing the job in the backend.
	// The execution can be synchronous or asynchronous, depending on the backend implementation.
	BidAccepted(context.Context, BidAcceptedRequest) (BidAcceptedResult, error)
	// BidRejected rejects a bid for a given executionID.
	BidRejected(context.Context, BidRejectedRequest) (BidRejectedResult, error)
	// ResultAccepted accepts a result for a given executionID, which will trigger publishing the result to the
	// destination specified in the job.
	ResultAccepted(context.Context, ResultAcceptedRequest) (ResultAcceptedResult, error)
	// ResultRejected rejects a result for a given executionID.
	ResultRejected(context.Context, ResultRejectedRequest) (ResultRejectedResult, error)
	// CancelJob cancels a job for a given executionID.
	CancelJob(context.Context, CancelJobRequest) (CancelJobResult, error)
}

type RequestMetadata struct {
	SourcePeerID    string
	DestPeerID      string
	SourceRequestID string
}
type AskForBidRequest struct {
	// Job specifies the job to be executed.
	Job model.Job
	// ShardIndexes specifies the shard indexes to be executed.
	// This enables the requester to ask for bids for a subset of the shards of a job.
	ShardIndexes []int
	RequestMetadata
}

type AskForBidResponse struct {
	ShardResponse []AskForBidShardResponse
}

type AskForBidShardResponse struct {
	ShardIndex  int
	Accepted    bool
	Reason      string
	ExecutionID string
}

type BidAcceptedRequest struct {
	ExecutionID string
	RequestMetadata
}

type BidAcceptedResult struct {
}

type BidRejectedRequest struct {
	ExecutionID   string
	Justification string
	RequestMetadata
}

type BidRejectedResult struct {
}

type ResultAcceptedRequest struct {
	ExecutionID string
	RequestMetadata
}

type ResultAcceptedResult struct {
}

type ResultRejectedRequest struct {
	ExecutionID   string
	Justification string
	RequestMetadata
}

type ResultRejectedResult struct {
}

type CancelJobRequest struct {
	ExecutionID   string
	Justification string
	RequestMetadata
}

type CancelJobResult struct {
}
