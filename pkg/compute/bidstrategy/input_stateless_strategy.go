package bidstrategy

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type StatelessJobStrategyParams struct {
	RejectStatelessJobs bool
}

type StatelessJobStrategy struct {
	rejectStatelessJobs bool
}

func NewStatelessJobStrategy(params StatelessJobStrategyParams) *StatelessJobStrategy {
	return &StatelessJobStrategy{
		rejectStatelessJobs: params.RejectStatelessJobs,
	}
}

func (s *StatelessJobStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	// skip bidding if no input data is provided, and policy is to reject stateless jobs
	if s.rejectStatelessJobs && len(request.Job.Spec.Inputs) == 0 {
		return BidStrategyResponse{ShouldBid: false, Reason: "stateless jobs not accepted"}, nil
	}

	return newShouldBidResponse(), nil
}

func (s *StatelessJobStrategy) ShouldBidBasedOnUsage(
	_ context.Context, _ BidStrategyRequest, _ model.ResourceUsageData) (BidStrategyResponse, error) {
	return newShouldBidResponse(), nil
}
