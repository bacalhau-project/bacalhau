package semantic

import (
	"context"
	"fmt"
	"math"
	"strconv"
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

func (s *TimeoutStrategy) ShouldBid(_ context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	if request.Job.Spec.Timeout <= 0 {
		return bidstrategy.NewShouldBidResponse(), nil
	}

	// Timeout will be multiplied by 1000000000 (time.Second) when it gets converted to a time.Duration (which is an int64 underneath),
	// so make sure that iit can fit into it.
	if request.Job.Spec.Timeout > float64(math.MaxInt64/int64(time.Second)) {
		timeout := strconv.FormatFloat(request.Job.Spec.Timeout, 'f', -1, sixtyFourBitFloat)
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("job timeout %s exceeds maximum possible value", timeout),
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

const sixtyFourBitFloat = 64
