package semantic

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
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

const (
	maxReason    = "accept jobs with timeout %v (the maximum allowed is %v)"
	minReason    = "accept jobs with timeout %v (the minimum allowed is %v)"
	bypassReason = "allow client %q to bypass timeout limits" //nolint:gosec
)

func (s *TimeoutStrategy) ShouldBid(_ context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	timeout := request.Job.Task().Timeouts.GetExecutionTimeout()
	if timeout <= 0 {
		return bidstrategy.NewBidResponse(true, minReason, timeout.String(), 0), nil
	}

	for _, clientID := range s.jobExecutionTimeoutClientIDBypassList {
		if request.Job.Namespace == clientID {
			return bidstrategy.NewBidResponse(true, bypassReason, clientID), nil
		}
	}

	// skip bidding if the job spec defined a timeout value higher or lower than what we are willing to accept
	if timeout < s.minJobExecutionTimeout {
		return bidstrategy.NewBidResponse(false, minReason, timeout.String(), s.minJobExecutionTimeout.String()), nil
	}

	success := s.maxJobExecutionTimeout <= 0 || (s.maxJobExecutionTimeout > 0 && timeout <= s.maxJobExecutionTimeout)
	return bidstrategy.NewBidResponse(success, maxReason, timeout.String(), s.maxJobExecutionTimeout.String()), nil
}
