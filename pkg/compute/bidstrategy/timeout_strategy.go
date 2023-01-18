package bidstrategy

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type TimeoutStrategyParams struct {
	MaxJobExecutionTimeout time.Duration
	MinJobExecutionTimeout time.Duration

	JobExecutionTimeoutClientIDBypassList []string
}

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

func (s *TimeoutStrategy) ShouldBid(_ context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	for _, clientID := range s.jobExecutionTimeoutClientIDBypassList {
		if request.Job.Metadata.ClientID == clientID {
			return newShouldBidResponse(), nil
		}
	}

	if request.Job.Spec.Timeout <= 0 {
		return newShouldBidResponse(), nil
	}
	// skip bidding if the job spec defined a timeout value higher or lower than what we are willing to accept
	if s.maxJobExecutionTimeout > 0 && request.Job.Spec.GetTimeout() > s.maxJobExecutionTimeout {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("job timeout %s exceeds maximum allowed %s", request.Job.Spec.GetTimeout(), s.maxJobExecutionTimeout),
		}, nil
	}
	if request.Job.Spec.GetTimeout() < s.minJobExecutionTimeout {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("job timeout %s below minimum allowed %s", request.Job.Spec.GetTimeout(), s.minJobExecutionTimeout),
		}, nil
	}
	return newShouldBidResponse(), nil
}

func (s *TimeoutStrategy) ShouldBidBasedOnUsage(
	_ context.Context, _ BidStrategyRequest, _ model.ResourceUsageData) (BidStrategyResponse, error) {
	return newShouldBidResponse(), nil
}
