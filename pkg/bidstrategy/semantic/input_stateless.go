package semantic

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
)

type StatelessJobStrategyParams struct {
	RejectStatelessJobs bool
}

// Compile-time check of interface implementation
var _ bidstrategy.SemanticBidStrategy = (*StatelessJobStrategy)(nil)

type StatelessJobStrategy struct {
	rejectStatelessJobs bool
}

func NewStatelessJobStrategy(params StatelessJobStrategyParams) *StatelessJobStrategy {
	return &StatelessJobStrategy{
		rejectStatelessJobs: params.RejectStatelessJobs,
	}
}

func (s *StatelessJobStrategy) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	// skip bidding if no input data is provided, and policy is to reject stateless jobs
	if s.rejectStatelessJobs && len(request.Job.Spec.Inputs) == 0 {
		return bidstrategy.BidStrategyResponse{ShouldBid: false, Reason: "stateless jobs not accepted"}, nil
	}

	return bidstrategy.NewShouldBidResponse(), nil
}
