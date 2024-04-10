package compute

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type BidderParams struct {
	NodeID           string
	SemanticStrategy []bidstrategy.SemanticBidStrategy
	ResourceStrategy []bidstrategy.ResourceBidStrategy
	UsageCalculator  capacity.UsageCalculator
	Store            store.ExecutionStore
	Executor         Executor
	Callback         Callback
	GetApproveURL    func() *url.URL
}

type Bidder struct {
	nodeID          string
	store           store.ExecutionStore
	usageCalculator capacity.UsageCalculator
	executor        Executor
	callback        Callback
	getApproveURL   func() *url.URL

	semanticStrategy []bidstrategy.SemanticBidStrategy
	resourceStrategy []bidstrategy.ResourceBidStrategy
}

func NewBidder(params BidderParams) Bidder {
	return Bidder{
		nodeID:           params.NodeID,
		store:            params.Store,
		usageCalculator:  params.UsageCalculator,
		getApproveURL:    params.GetApproveURL,
		executor:         params.Executor,
		callback:         params.Callback,
		semanticStrategy: params.SemanticStrategy,
		resourceStrategy: params.ResourceStrategy,
	}
}

func (b Bidder) ReturnBidResult(
	ctx context.Context,
	localExecutionState store.LocalExecutionState,
	response *bidstrategy.BidStrategyResponse,
) {
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
		Event:             RespondedToBidEvent(response),
	}
	b.callback.OnBidComplete(ctx, result)
}

type BidderRequest struct {
	SourcePeerID string
	// Execution specifies the job to be executed.
	Execution *models.Execution
	// WaitForApproval specifies whether the compute node should wait for the requester to approve the bid.
	// if set to true, the compute node will not start the execution until the requester approves the bid.
	// If set to false, the compute node will automatically start the execution after bidding and when resources are available.
	WaitForApproval bool

	// ResourceUsage specifies the requested resources for this execution
	ResourceUsage *models.Resources
}

// TODO: evaluate the need for async bidding and marking bids as waiting
// https://github.com/bacalhau-project/bacalhau/issues/3732
func (b Bidder) RunBidding(ctx context.Context, bidRequest *BidderRequest) {
	routingMetadata := RoutingMetadata{
		// the source of this response is the bidders nodeID.
		SourcePeerID: b.nodeID,
		// the target of this response is the source of the request.
		TargetPeerID: bidRequest.SourcePeerID,
	}
	executionMetadata := ExecutionMetadata{
		ExecutionID: bidRequest.Execution.ID,
		JobID:       bidRequest.Execution.JobID,
	}

	job := bidRequest.Execution.Job

	bidResult, err := b.doBidding(ctx, job, bidRequest.ResourceUsage)
	if err != nil {
		b.callback.OnComputeFailure(ctx, ComputeError{
			RoutingMetadata:   routingMetadata,
			ExecutionMetadata: executionMetadata,
			Event:             models.EventFromError(models.EventTopicExecutionBidding, err),
		})
		return
	}
	b.handleBidResult(ctx, bidResult, bidRequest.SourcePeerID, bidRequest.WaitForApproval, bidRequest.Execution)
}

type bidStrategyResponse struct {
	bid                 bool
	wait                bool
	reason              string
	calculatedResources *models.Resources
}

func (b Bidder) handleBidResult(
	ctx context.Context,
	result *bidStrategyResponse,
	targetPeer string,
	waitForApproval bool,
	execution *models.Execution,
) {
	var (
		routingMetadata = RoutingMetadata{
			// the source of this response is the bidders nodeID.
			SourcePeerID: b.nodeID,
			// the target of this response is the source of the request.
			TargetPeerID: targetPeer,
		}
		executionMetadata = ExecutionMetadata{
			ExecutionID: execution.ID,
			JobID:       execution.JobID,
		}
		handleComputeFailure = func(ctx context.Context, err error, reason string) {
			log.Ctx(ctx).WithLevel(logger.ErrOrDebug(err)).Err(err).Msg(reason)
			if err == nil {
				err = errors.New(reason)
			}
			b.callback.OnComputeFailure(ctx, ComputeError{
				RoutingMetadata:   routingMetadata,
				ExecutionMetadata: executionMetadata,
				Event:             models.EventFromError(models.EventTopicExecutionBidding, err),
			})
		}
		handleBidComplete = func(ctx context.Context, result *bidStrategyResponse) {
			b.callback.OnBidComplete(ctx, BidResult{
				RoutingMetadata:   routingMetadata,
				ExecutionMetadata: executionMetadata,
				Accepted:          result.bid,
				Wait:              result.wait,
				Event: RespondedToBidEvent(&bidstrategy.BidStrategyResponse{
					ShouldBid:  result.bid,
					ShouldWait: result.wait,
					Reason:     result.reason,
				}),
			})
		}
	)

	if !waitForApproval {
		if !result.bid || result.wait {
			handleComputeFailure(ctx, nil, fmt.Sprintf("job rejected: %s", result.reason))
			return
		}

		execution.AllocateResources(execution.Job.Task().Name, *result.calculatedResources)
		localExecution := store.NewLocalExecutionState(execution, targetPeer)
		localExecution.State = store.ExecutionStateBidAccepted

		if err := b.store.CreateExecution(ctx, *localExecution); err != nil {
			handleComputeFailure(ctx, err, "failed to create execution state")
			return
		}
		if err := b.executor.Run(ctx, *localExecution); err != nil {
			// no need to check for run errors as they are already handled by the executor.
			log.Ctx(ctx).Error().Err(err).Msg("failed to run execution")
			return
		}
		return
	}

	// if we are bidding or waiting create an execution
	if result.bid || result.wait {
		execution.AllocateResources(execution.Job.Task().Name, *result.calculatedResources)
		localExecution := store.NewLocalExecutionState(execution, targetPeer)
		if err := b.store.CreateExecution(ctx, *localExecution); err != nil {
			handleComputeFailure(ctx, err, "failed to create execution state")
			return
		}
	}

	// if we are not bidding and not wait return a response, we can't do this job. mark as complete then bail
	if !result.bid && !result.wait {
		handleBidComplete(ctx, result)
		return
	}

	// were not waiting return a response.
	if !result.wait {
		handleBidComplete(ctx, result)
	}
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
	job *models.Job,
	resourceUsage *models.Resources) (*bidStrategyResponse, error) {
	// NB(forrest): allways run semantic bidding before resource bidding since generally there isn't much point in
	// calling resource strategies that require DiskUsageCalculator.Calculate (a precursor to checking bidding) if
	// semantically the job cannot run.
	semanticResponse, err := b.runSemanticBidding(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("semantic bidding: %w", err)
	}

	// we shouldn't bid, and we're not waiting, bail.
	if !semanticResponse.bid && !semanticResponse.wait {
		return semanticResponse, nil
	}

	// else the request is semantically biddable or waiting, calculate resource usage and check resource-based bidding.
	resourceResponse, err := b.runResourceBidding(ctx, job, resourceUsage)
	if err != nil {
		return nil, fmt.Errorf("resource bidding: %w", err)
	}

	return &bidStrategyResponse{
		bid:                 resourceResponse.bid,
		wait:                semanticResponse.wait || resourceResponse.wait,
		reason:              resourceResponse.reason,
		calculatedResources: resourceResponse.calculatedResources,
	}, nil
}

func (b Bidder) runSemanticBidding(
	ctx context.Context,
	job *models.Job,
) (*bidStrategyResponse, error) {
	// ask the bidding strategy if we should bid on this job
	request := bidstrategy.BidStrategyRequest{
		NodeID:   b.nodeID,
		Job:      *job,
		Callback: b.getApproveURL(),
	}

	// assume we are bidding unless a request is rejected
	shouldBid := true
	// assume we're not waiting to bid unless a request indicates so.
	shouldWait := false
	reasons := make([]string, 0, len(b.semanticStrategy))
	for _, s := range b.semanticStrategy {
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
			reasons = append(reasons, fmt.Sprintf("waiting to bid: %s", resp.Reason))
		} else if !resp.ShouldBid {
			shouldBid = false
			reasons = append(reasons, fmt.Sprintf("rejected bid: %s", resp.Reason))
		}
	}

	return &bidStrategyResponse{
		bid:    shouldBid,
		wait:   shouldWait,
		reason: strings.Join(reasons, "; "),
	}, nil
}

func (b Bidder) runResourceBidding(
	ctx context.Context,
	job *models.Job,
	resources *models.Resources,
) (*bidStrategyResponse, error) {
	// calculate resource usage of the job, failure here represents a compute failure.
	resourceUsage, err := b.usageCalculator.Calculate(ctx, *job, *resources)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error calculating resource requirements for job")
		return nil, fmt.Errorf("calculating resource usage of job: %w", err)
	}

	// ask the bidding strategy if we should bid on this job
	request := bidstrategy.BidStrategyRequest{
		NodeID:   b.nodeID,
		Job:      *job,
		Callback: b.getApproveURL(),
	}

	// assume we are bidding unless a request is rejected
	shouldBid := true
	// assume we're not waiting to bid unless a request indicates so.
	shouldWait := false
	reasons := make([]string, 0, len(b.resourceStrategy))

	// TODO(forrest): this can be parallelized with a wait group, room for improvement here if resource validation
	//  is time consuming.
	for _, s := range b.resourceStrategy {
		strategyType := reflect.TypeOf(s).String()
		resp, err := s.ShouldBidBasedOnUsage(ctx, request, *resourceUsage)
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
			reasons = append(reasons, fmt.Sprintf("waiting to bid: %s", resp.Reason))
		} else if !resp.ShouldBid {
			shouldBid = false
			reasons = append(reasons, fmt.Sprintf("rejected bid: %s", resp.Reason))
		}
	}

	return &bidStrategyResponse{
		bid:                 shouldBid,
		wait:                shouldWait,
		reason:              strings.Join(reasons, "; "),
		calculatedResources: resourceUsage,
	}, nil
}
