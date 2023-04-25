package compute

import (
	"context"
	"fmt"
	"net/url"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type BidderParams struct {
	NodeID           string
	SemanticStrategy bidstrategy.SemanticBidStrategy
	ResourceStrategy bidstrategy.ResourceBidStrategy
	Store            store.ExecutionStore
	Callback         Callback
	GetApproveURL    func() *url.URL
}

type Bidder struct {
	nodeID        string
	store         store.ExecutionStore
	callback      Callback
	getApproveURL func() *url.URL

	semanticStrategy bidstrategy.SemanticBidStrategy
	resourceStrategy bidstrategy.ResourceBidStrategy
}

func NewBidder(params BidderParams) Bidder {
	return Bidder{
		nodeID:           params.NodeID,
		store:            params.Store,
		getApproveURL:    params.GetApproveURL,
		callback:         params.Callback,
		semanticStrategy: params.SemanticStrategy,
		resourceStrategy: params.ResourceStrategy,
	}
}

func (b Bidder) RunBidding(ctx context.Context, request AskForBidRequest, usageCalc capacity.UsageCalculator) {
	var (
		// ask the bidding strategy if we should bid on this job
		bidStrategyRequest = bidstrategy.BidStrategyRequest{
			NodeID:   b.nodeID,
			Job:      request.Job,
			Callback: b.getApproveURL(),
		}

		routingMetadata = RoutingMetadata{
			// the source of this response is the bidders nodeID.
			SourcePeerID: b.nodeID,
			// the target of this response is the source of the request.
			TargetPeerID: request.SourcePeerID,
		}
		executionMetadata = ExecutionMetadata{
			ExecutionID: request.ExecutionID,
			JobID:       request.JobID,
		}
	)

	response, resourceUsage, err := b.doBidding(ctx, bidStrategyRequest, usageCalc)
	if err != nil {
		b.callback.OnComputeFailure(ctx, ComputeError{
			RoutingMetadata:   routingMetadata,
			ExecutionMetadata: executionMetadata,
			Err:               err.Error(),
		})

		log.Ctx(ctx).Error().Err(err).Msg("Error running bid strategy")
		return
	}

	result := BidResult{
		RoutingMetadata:   routingMetadata,
		ExecutionMetadata: executionMetadata,
		Accepted:          response.ShouldBid,
		Reason:            response.Reason,
	}

	// if we are not bidding and not wait return a response, we can't do this job. mark as complete then bail
	if !response.ShouldBid && !response.ShouldWait {
		b.callback.OnBidComplete(ctx, result)
		return
	}

	// if we are bidding or waiting create an execution
	if response.ShouldWait || response.ShouldBid {
		execution := store.NewExecution(request.ExecutionID, request.Job, request.SourcePeerID, *resourceUsage)
		if err := b.store.CreateExecution(ctx, *execution); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Unable to create execution state")
			return
		}
	}

	// were not waiting return a response.
	if !response.ShouldWait {
		b.callback.OnBidComplete(ctx, result)
	}
}

func (b Bidder) ReturnBidResult(ctx context.Context, execution store.Execution, response *bidstrategy.BidStrategyResponse) {
	if response.ShouldWait {
		return
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

// doBidding returns a response based on the below semantics. It should never be the case that semantic or resource
// strategies return `true` for both ShouldBid and ShouldWait. The last row is a special optimization case since if
// semantic bidding states we should not bid and not wait when the resource strategy will never be evaluated.
// We will always wait if at least one strategy specifies it. We will only bid if both strategies specify it.
// | SemanticShouldBid | SemanticShouldWait | ResourceShouldBid | ResourceShouldWait | ShouldBid | ShouldWait |
// |      true         |      false         |      true         |      false         |   true    |   false    |
// |      false        |      true          |      true         |      false         |   false   |   true     |
// |      true         |      false         |      false        |      true          |   false   |   true     |
// |      false        |      true          |      true         |      false         |   false   |   true     |
// |      false        |      true          |      false        |      false         |   false   |   true     |
// |      true         |      false         |      false        |      false         |   false   |   false    |
// |      false        |      false         |       N/A         |       N/A          |   false   |   false    |
func (b Bidder) doBidding(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	calculator capacity.UsageCalculator,
) (*bidstrategy.BidStrategyResponse, *model.ResourceUsageData, error) {

	// Check semantic bidding strategies before calculating resource usage.
	semanticResponse, err := b.semanticStrategy.ShouldBid(ctx, request)
	if err != nil {
		return nil, nil, fmt.Errorf("error asking bidding strategy if we should bid: %w", err)
	}

	// we shouldn't bid, and we're not waiting, bail.
	if !semanticResponse.ShouldBid && !semanticResponse.ShouldWait {
		return &semanticResponse, nil, nil
	}

	// the request is semantically biddable or waiting, calculate resource usage and check resource-based bidding.
	resourceUsage, err := calculator.Calculate(ctx, request.Job, capacity.ParseResourceUsageConfig(request.Job.Spec.Resources))
	if err != nil {
		return nil, nil, fmt.Errorf("error calculating resource requirements for job: %w", err)
	}
	resourceResponse, err := b.resourceStrategy.ShouldBidBasedOnUsage(ctx, request, resourceUsage)
	if err != nil {
		return nil, nil, fmt.Errorf("error asking bidding strategy if we should bid: %w", err)
	}

	return &bidstrategy.BidStrategyResponse{
		ShouldBid:  resourceResponse.ShouldBid,
		ShouldWait: semanticResponse.ShouldWait || resourceResponse.ShouldWait,
		Reason:     "",
	}, &resourceUsage, nil

}
