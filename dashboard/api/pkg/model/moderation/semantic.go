package moderation

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type semanticBidModerator struct {
	strategy bidstrategy.BidStrategy
	onWait   ExecutionModerator
}

func NewSemanticBidModerator(strategy bidstrategy.BidStrategy, onWait ExecutionModerator) ExecutionModerator {
	return &semanticBidModerator{strategy: strategy, onWait: onWait}
}

// ShouldExecute implements ExecutionModerator
func (moderator *semanticBidModerator) ShouldExecute(
	ctx context.Context,
	probe *bidstrategy.JobSelectionPolicyProbeData,
) (*bidstrategy.BidStrategyResponse, error) {
	bidResponse, err := moderator.strategy.ShouldBid(ctx, bidstrategy.BidStrategyRequest{
		NodeID: probe.NodeID,
		Job: model.Job{
			Metadata: model.Metadata{ID: probe.JobID},
			Spec:     probe.Spec,
		},
		Callback: probe.Callback,
	})

	if err == nil && bidResponse.ShouldWait {
		return moderator.onWait.ShouldExecute(ctx, probe)
	} else {
		// We can respond immediately.
		return &bidResponse, err
	}
}

var _ ExecutionModerator = (*semanticBidModerator)(nil)
