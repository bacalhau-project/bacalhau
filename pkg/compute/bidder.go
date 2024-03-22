package compute

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type BidderParams struct {
	NodeID           string
	SemanticStrategy []bidstrategy.SemanticBidStrategy
	ResourceStrategy []bidstrategy.ResourceBidStrategy
	Store            store.ExecutionStore
	Executor         Executor
	Callback         Callback
	GetApproveURL    func() *url.URL
}

type Bidder struct {
	nodeID        string
	store         store.ExecutionStore
	executor      Executor
	callback      Callback
	getApproveURL func() *url.URL

	semanticStrategy []bidstrategy.SemanticBidStrategy
	resourceStrategy []bidstrategy.ResourceBidStrategy
}

func NewBidder(params BidderParams) Bidder {
	return Bidder{
		nodeID:           params.NodeID,
		store:            params.Store,
		getApproveURL:    params.GetApproveURL,
		executor:         params.Executor,
		callback:         params.Callback,
		semanticStrategy: params.SemanticStrategy,
		resourceStrategy: params.ResourceStrategy,
	}
}

// TODO: evaluate the need for async bidding and marking bids as waiting
func (b Bidder) RunBidding(ctx context.Context, request AskForBidRequest, resources *models.Resources) (*BidResult, error) {
	var (
		// ask the bidding strategy if we should bid on this job
		bidStrategyRequest = bidstrategy.BidStrategyRequest{
			NodeID:   b.nodeID,
			Job:      *request.Execution.Job,
			Callback: b.getApproveURL(),
		}
	)

	// run bidding
	response, err := b.doBidding(ctx, bidStrategyRequest, resources)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("running bid strategy")
		return nil, fmt.Errorf("running bid strategy: %w", err)
	}

	return &BidResult{
		RoutingMetadata: RoutingMetadata{
			// the source of this response is the bidders nodeID.
			SourcePeerID: b.nodeID,
			// the target of this response is the source of the request.
			TargetPeerID: request.SourcePeerID,
		},
		ExecutionMetadata: ExecutionMetadata{
			ExecutionID: request.Execution.ID,
			JobID:       request.Execution.JobID,
		},
		Accepted: response.ShouldBid,
		Wait:     response.ShouldWait,
		Reason:   response.Reason,
	}, nil
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
	resourceUsage *models.Resources) (*bidstrategy.BidStrategyResponse, error) {
	// Check semantic bidding strategies before calculating resource usage.
	semanticResponse, err := runSemanticBidding(ctx, request, b.semanticStrategy...)
	if err != nil {
		return nil, fmt.Errorf("semantic bidding: %w", err)
	}

	// we shouldn't bid, and we're not waiting, bail.
	if !semanticResponse.ShouldBid && !semanticResponse.ShouldWait {
		return semanticResponse, nil
	}

	// else the request is semantically biddable or waiting, calculate resource usage and check resource-based bidding.
	resourceResponse, err := runResourceBidding(ctx, request, resourceUsage, b.resourceStrategy...)
	if err != nil {
		return nil, fmt.Errorf("resource bidding: %w", err)
	}

	return &bidstrategy.BidStrategyResponse{
		ShouldBid:  resourceResponse.ShouldBid,
		ShouldWait: semanticResponse.ShouldWait || resourceResponse.ShouldWait,
		Reason:     resourceResponse.Reason,
	}, nil
}

func runSemanticBidding(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	strategies ...bidstrategy.SemanticBidStrategy,
) (*bidstrategy.BidStrategyResponse, error) {
	// assume we are bidding unless a request is rejected
	shouldBid := true
	// assume we're not waiting to bid unless a request indicates so.
	shouldWait := false
	reasons := make([]string, 0, len(strategies))
	for _, s := range strategies {
		// TODO(forrest): this can be parallelized with a wait group, although semantic checks ought to be quick.
		strategyType := reflect.TypeOf(s).String()
		resp, err := s.ShouldBid(ctx, request)
		if err != nil {
			errMsg := fmt.Sprintf("bid strategy: %s failed", strategyType)
			log.Ctx(ctx).Error().Err(err).Msgf(errMsg)
			// NB: failure here results in a callback to OnComputeFailure
			return nil, errors.Wrap(err, errMsg)
		}
		log.Ctx(ctx).Info().
			Str("Job", request.Job.ID).
			Str("strategy", strategyType).
			Bool("bid", resp.ShouldBid).
			Bool("wait", resp.ShouldWait).
			Str("reason", resp.Reason).
			Msgf("bit strategy response")

		if resp.ShouldWait {
			shouldWait = true
			reasons = append(reasons, fmt.Sprintf("%s: waiting to bid: %s",
				strategyType, resp.Reason))
		} else if !resp.ShouldBid {
			shouldBid = false
			reasons = append(reasons, fmt.Sprintf("%s: rejected bid: %s",
				strategyType, resp.Reason))
		}
	}

	return &bidstrategy.BidStrategyResponse{
		ShouldBid:  shouldBid,
		ShouldWait: shouldWait,
		Reason:     strings.Join(reasons, "; "),
	}, nil
}

func runResourceBidding(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	resources *models.Resources,
	strategies ...bidstrategy.ResourceBidStrategy,
) (*bidstrategy.BidStrategyResponse, error) {
	// assume we are bidding unless a request is rejected
	shouldBid := true
	// assume we're not waiting to bid unless a request indicates so.
	shouldWait := false
	reasons := make([]string, 0, len(strategies))
	// TODO(forrest): this can be parallelized with a wait group, room for improvement here if resource validation
	//  is time consuming.
	for _, s := range strategies {
		strategyType := reflect.TypeOf(s).String()
		resp, err := s.ShouldBidBasedOnUsage(ctx, request, *resources)
		if err != nil {
			errMsg := fmt.Sprintf("bid strategy: %s failed", strategyType)
			log.Ctx(ctx).Error().Err(err).Msgf(errMsg)
			// NB: failure here results in a callback to OnComputeFailure
			return nil, errors.Wrap(err, errMsg)
		}
		log.Ctx(ctx).Info().
			Str("Job", request.Job.ID).
			Str("strategy", strategyType).
			Bool("bid", resp.ShouldBid).
			Bool("wait", resp.ShouldWait).
			Str("reason", resp.Reason).
			Msgf("bit strategy response")

		if resp.ShouldWait {
			shouldWait = true
			reasons = append(reasons, fmt.Sprintf("%s: waiting to bid: %s",
				strategyType, resp.Reason))
		} else if !resp.ShouldBid {
			shouldBid = false
			reasons = append(reasons, fmt.Sprintf("%s: rejected bid: %s",
				strategyType, resp.Reason))
		}
	}

	return &bidstrategy.BidStrategyResponse{
		ShouldBid:  shouldBid,
		ShouldWait: shouldWait,
		Reason:     strings.Join(reasons, ";"),
	}, nil

}

func (b Bidder) ReturnBidResult(
	ctx context.Context, localExecutionState store.LocalExecutionState, response *bidstrategy.BidStrategyResponse) {
	if response.ShouldWait {
		return
	}
	result := BidResult{
		RoutingMetadata: RoutingMetadata{
			SourcePeerID: b.nodeID,
			TargetPeerID: localExecutionState.RequesterNodeID,
		},
		ExecutionMetadata: NewExecutionMetadata(localExecutionState.Execution),
		Accepted:          response.ShouldBid,
		Reason:            response.Reason,
	}
	b.callback.OnBidComplete(ctx, result)
}
