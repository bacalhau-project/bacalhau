package compute

import (
	"context"
	"fmt"
	"net/url"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

type BidderParams struct {
	NodeID        string
	Strategy      bidstrategy.BidStrategy
	Store         store.ExecutionStore
	Callback      Callback
	GetApproveURL func() *url.URL
}

type Bidder struct {
	nodeID        string
	strategy      bidstrategy.BidStrategy
	store         store.ExecutionStore
	callback      Callback
	getApproveURL func() *url.URL
}

func NewBidder(params BidderParams) Bidder {
	return Bidder{
		nodeID:        params.NodeID,
		strategy:      params.Strategy,
		store:         params.Store,
		getApproveURL: params.GetApproveURL,
		callback:      params.Callback,
	}
}

func (b Bidder) RunBidding(ctx context.Context, execution store.Execution) {
	// ask the bidding strategy if we should bid on this job
	bidStrategyRequest := bidstrategy.BidStrategyRequest{
		NodeID:   b.nodeID,
		Job:      execution.Job,
		Callback: b.getApproveURL(),
	}

	response, err := b.doBidding(ctx, bidStrategyRequest, execution.ResourceUsage)
	if err != nil {
		// TODO what do we do with it?
		log.Ctx(ctx).Error().Err(err).Msg("Error running bid strategy")
	}

	b.ReturnBidResult(ctx, execution, response)
}

func (b Bidder) ReturnBidResult(ctx context.Context, execution store.Execution, response *bidstrategy.BidStrategyResponse) {
	if response.ShouldWait {
		return
	}

	if !response.ShouldBid {
		err := b.store.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
			ExecutionID:   execution.ID,
			NewState:      store.ExecutionStateCancelled,
			ExpectedState: store.ExecutionStateCreated,
			Comment:       response.Reason,
		})

		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Unable to update execution state")
			return
		}
	}

	result := BidResult{
		RoutingMetadata: RoutingMetadata{
			SourcePeerID: b.nodeID,
			TargetPeerID: execution.RequesterNodeID,
		},
		ExecutionMetadata: NewExecutionMetadata(execution),
		Accepted:          response.ShouldBid,
		Reason:            response.Reason,
	}
	b.callback.OnBidComplete(ctx, result)
}

func (b Bidder) doBidding(
	ctx context.Context,
	bidStrategyRequest bidstrategy.BidStrategyRequest,
	jobRequirements model.ResourceUsageData,
) (*bidstrategy.BidStrategyResponse, error) {
	// Check bidding strategies before having to calculate resource usage
	bidStrategyResponse, err := b.strategy.ShouldBid(ctx, bidStrategyRequest)
	if err != nil {
		return nil, fmt.Errorf("error asking bidding strategy if we should bid: %w", err)
	}

	if bidStrategyResponse.ShouldBid {
		// Check bidding strategies after calculating resource usage
		bidStrategyResponse, err = b.strategy.ShouldBidBasedOnUsage(ctx, bidStrategyRequest, jobRequirements)
		if err != nil {
			return nil, fmt.Errorf("error asking bidding strategy if we should bid: %w", err)
		}
	}

	return &bidStrategyResponse, nil
}
