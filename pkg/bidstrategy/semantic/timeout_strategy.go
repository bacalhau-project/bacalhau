package semantic

import (
	"context"
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

const (
	maxReason    = "accept jobs with timeout %v (the maximum allowed is %v)"
	minReason    = "accept jobs with timeout %v (the minimum allowed is %v)"
	bypassReason = "allow client %q to bypass timeout limits" //nolint:gosec
)

func (s *TimeoutStrategy) ShouldBid(_ context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	timeoutSeconds := request.Job.Task().Timeouts.ExecutionTimeout
	if timeoutSeconds <= 0 {
		return bidstrategy.NewBidResponse(true, minReason, timeoutSeconds, 0), nil
	}

	// Timeout will be multiplied by 1000000000 (time.Second) when it gets
	// converted to a time.Duration (which is an int64 underneath), so make sure
	// that it can fit into it.
	var maxTimeout = int64(model.NoJobTimeout.Seconds())
	if request.Job.Task().Timeouts.ExecutionTimeout > maxTimeout {
		return bidstrategy.NewBidResponse(false, maxReason, timeoutSeconds, maxTimeout), nil
	}

	for _, clientID := range s.jobExecutionTimeoutClientIDBypassList {
		if request.Job.Namespace == clientID {
			return bidstrategy.NewBidResponse(true, bypassReason, clientID), nil
		}
	}

	// skip bidding if the job spec defined a timeout value higher or lower than what we are willing to accept
	timeoutDuration := request.Job.Task().Timeouts.GetExecutionTimeout()
	if timeoutDuration < s.minJobExecutionTimeout {
		return bidstrategy.NewBidResponse(false, minReason, timeoutDuration.String(), s.minJobExecutionTimeout.String()), nil
	}

	success := s.maxJobExecutionTimeout <= 0 || (s.maxJobExecutionTimeout > 0 && timeoutDuration <= s.maxJobExecutionTimeout)
	return bidstrategy.NewBidResponse(success, maxReason, timeoutDuration.String(), s.maxJobExecutionTimeout.String()), nil
}
