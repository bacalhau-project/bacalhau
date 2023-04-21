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
	go func() {
		// ask the bidding strategy if we should bid on this job
		bidStrategyRequest := bidstrategy.BidStrategyRequest{
			NodeID:   b.nodeID,
			Job:      request.Job,
			Callback: b.getApproveURL(),
		}

		response, resourceUsage, err := b.doBidding(ctx, bidStrategyRequest, usageCalc)
		if err != nil {
			b.callback.OnComputeFailure(ctx, ComputeError{
				RoutingMetadata: RoutingMetadata{
					// TODO double check these.
					SourcePeerID: b.nodeID,
					TargetPeerID: request.SourcePeerID,
				},
				ExecutionMetadata: ExecutionMetadata{
					ExecutionID: request.ExecutionID,
					JobID:       request.JobID,
				},
				Err: err.Error(),
			})

			log.Ctx(ctx).Error().Err(err).Msg("Error running bid strategy")
			return
		}

		result := BidResult{
			RoutingMetadata: RoutingMetadata{
				SourcePeerID: b.nodeID,
				TargetPeerID: request.SourcePeerID,
			},
			ExecutionMetadata: ExecutionMetadata{
				ExecutionID: request.ExecutionID,
				JobID:       request.JobID,
			},
			Accepted: response.ShouldBid,
			Reason:   response.Reason,
		}

		// if we are not bidding and not wait return a response, we can't do this job.
		if !response.ShouldBid && !response.ShouldWait {
			b.callback.OnBidComplete(ctx, result)
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
	}()
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
	calculator capacity.UsageCalculator,
) (*bidstrategy.BidStrategyResponse, *model.ResourceUsageData, error) {

	// Check bidding strategies before having to calculate resource usage
	bidStrategyResponse, err := b.semanticStrategy.ShouldBid(ctx, bidStrategyRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("error asking bidding strategy if we should bid: %w", err)
	}

	var resourceUsage model.ResourceUsageData
	if bidStrategyResponse.ShouldBid {
		resourceUsage, err = calculator.Calculate(ctx, bidStrategyRequest.Job, capacity.ParseResourceUsageConfig(bidStrategyRequest.Job.Spec.Resources))
		if err != nil {
			return nil, nil, fmt.Errorf("error calculating resource requirements for job: %w", err)
		}
		// Check bidding strategies after calculating resource usage
		bidStrategyResponse, err = b.resourceStrategy.ShouldBidBasedOnUsage(ctx, bidStrategyRequest, resourceUsage)
		if err != nil {
			return nil, nil, fmt.Errorf("error asking bidding strategy if we should bid: %w", err)
		}
	}

	return &bidStrategyResponse, &resourceUsage, nil
}
