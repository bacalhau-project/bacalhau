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

const (
	statelessReason = "accept jobs without any input volumes"
	statefulReason  = "accept jobs with input volumes"
)

func (s *StatelessJobStrategy) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	// skip bidding if no input data is provided, and policy is to reject stateless jobs
	if len(request.Job.Task().InputSources) > 0 {
		return bidstrategy.NewBidResponse(true, statefulReason), nil
	} else {
		return bidstrategy.NewBidResponse(!s.rejectStatelessJobs, statelessReason), nil
	}
}
