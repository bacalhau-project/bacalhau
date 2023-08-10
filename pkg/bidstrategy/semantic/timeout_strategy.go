package semantic

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type TimeoutStrategyParams struct {
	MaxJobExecutionTimeout time.Duration
	MinJobExecutionTimeout time.Duration

	JobExecutionTimeoutClientIDBypassList []string
}

var _ bidstrategy.SemanticBidStrategy = (*TimeoutStrategy)(nil)

type TimeoutStrategy struct {
	maxJobExecutionTimeout                time.Duration
	minJobExecutionTimeout                time.Duration
	jobExecutionTimeoutClientIDBypassList []string
}

func NewTimeoutStrategy(params TimeoutStrategyParams) *TimeoutStrategy {
	return &TimeoutStrategy{
		maxJobExecutionTimeout:                params.MaxJobExecutionTimeout,
		minJobExecutionTimeout:                params.MinJobExecutionTimeout,
		jobExecutionTimeoutClientIDBypassList: params.JobExecutionTimeoutClientIDBypassList,
	}
}

func (s *TimeoutStrategy) ShouldBid(_ context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	if request.Job.Spec.Timeout <= 0 {
		return bidstrategy.NewShouldBidResponse(), nil
	}

	// Timeout will be multiplied by 1000000000 (time.Second) when it gets
	// converted to a time.Duration (which is an int64 underneath), so make sure
	// that it can fit into it.
	var maxTimeout = int64(model.NoJobTimeout.Seconds())
	if request.Job.Spec.Timeout > maxTimeout {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("job timeout %d exceeds maximum possible value %d", request.Job.Spec.Timeout, maxTimeout),
		}, nil
	}

	for _, clientID := range s.jobExecutionTimeoutClientIDBypassList {
		if request.Job.Metadata.ClientID == clientID {
			return bidstrategy.NewShouldBidResponse(), nil
		}
	}

	// skip bidding if the job spec defined a timeout value higher or lower than what we are willing to accept
	if s.maxJobExecutionTimeout > 0 && request.Job.Spec.GetTimeout() > s.maxJobExecutionTimeout {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("job timeout %s exceeds maximum allowed %s", request.Job.Spec.GetTimeout(), s.maxJobExecutionTimeout),
		}, nil
	}
	if request.Job.Spec.GetTimeout() < s.minJobExecutionTimeout {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("job timeout %s below minimum allowed %s", request.Job.Spec.GetTimeout(), s.minJobExecutionTimeout),
		}, nil
	}
	return bidstrategy.NewShouldBidResponse(), nil
}
