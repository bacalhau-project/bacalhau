package frontend

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type Service interface {
	AskForBid(context.Context, AskForBidRequest) (AskForBidResponse, error)
	BidAccepted(context.Context, BidAcceptedRequest) (BidAcceptedResult, error)
	BidRejected(context.Context, BidRejectedRequest) (BidRejectedResult, error)
	ResultAccepted(context.Context, ResultAcceptedRequest) (ResultAcceptedResult, error)
	ResultRejected(context.Context, ResultRejectedRequest) (ResultRejectedResult, error)
	CancelJob(context.Context, CancelJobRequest) (CancelJobResult, error)
}

type AskForBidRequest struct {
	Job          model.Job
	ShardIndexes []int
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
}

type BidAcceptedResult struct {
}

type BidRejectedRequest struct {
	ExecutionID   string
	Justification string
}

type BidRejectedResult struct {
}

type ResultAcceptedRequest struct {
	ExecutionID string
}

type ResultAcceptedResult struct {
}

type ResultRejectedRequest struct {
	ExecutionID   string
	Justification string
}

type ResultRejectedResult struct {
}

type CancelJobRequest struct {
	ExecutionID   string
	Justification string
}

type CancelJobResult struct {
}
