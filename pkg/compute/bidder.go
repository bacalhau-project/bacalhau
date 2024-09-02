package compute

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type bidStrategyResponse struct {
	bid                 bool
	reason              string
	calculatedResources *models.Resources
}

type BidderParams struct {
	SemanticStrategy []bidstrategy.SemanticBidStrategy
	ResourceStrategy []bidstrategy.ResourceBidStrategy
	UsageCalculator  capacity.UsageCalculator
	Store            store.ExecutionStore
}

type Bidder struct {
	store            store.ExecutionStore
	usageCalculator  capacity.UsageCalculator
	semanticStrategy []bidstrategy.SemanticBidStrategy
	resourceStrategy []bidstrategy.ResourceBidStrategy
}

func NewBidder(params BidderParams) Bidder {
	return Bidder{
		store:            params.Store,
		usageCalculator:  params.UsageCalculator,
		semanticStrategy: params.SemanticStrategy,
		resourceStrategy: params.ResourceStrategy,
	}
}

func (b Bidder) RunBidding(ctx context.Context, execution *models.Execution) {
	bidResult, err := b.doBidding(ctx, execution.Job)
	if err != nil {
		b.handleError(ctx, execution, err)
		return
	}
	b.handleBidResult(ctx, execution, bidResult)
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
func (b Bidder) doBidding(ctx context.Context, job *models.Job) (*bidStrategyResponse, error) {
	// NB(forrest): allways run semantic bidding before resource bidding since generally there isn't much point in
	// calling resource strategies that require DiskUsageCalculator.Calculate (a precursor to checking bidding) if
	// semantically the job cannot run.
	semanticResponse, err := b.runSemanticBidding(ctx, job)
	if err != nil {
		return nil, err
	}

	// we shouldn't bid, and we're not waiting, bail.
	if !semanticResponse.bid {
		return semanticResponse, nil
	}

	// else the request is semantically biddable or waiting, calculate resource usage and check resource-based bidding.
	resourceResponse, err := b.runResourceBidding(ctx, job)
	if err != nil {
		return nil, err
	}

	return resourceResponse, nil
}

func (b Bidder) runSemanticBidding(ctx context.Context, job *models.Job) (*bidStrategyResponse, error) {
	// ask the bidding strategy if we should bid on this job
	request := bidstrategy.BidStrategyRequest{
		Job: *job,
	}

	// assume we are bidding unless a request is rejected
	shouldBid := true
	reasons := make([]string, 0, len(b.semanticStrategy))
	for _, s := range b.semanticStrategy {
		// TODO(forrest): this can be parallelized with a wait group, although semantic checks ought to be quick.
		strategyType := reflect.TypeOf(s).String()
		resp, err := s.ShouldBid(ctx, request)

		if err != nil || !resp.ShouldBid {
			log.Ctx(ctx).WithLevel(logger.ErrOrDebug(err)).
				Err(err).
				Str("Job", request.Job.ID).
				Str("Strategy", strategyType).
				Bool("Bid", resp.ShouldBid).
				Str("Reason", resp.Reason).
				Send()
		}

		if err != nil {
			// NB: failure here results in a callback to OnComputeFailure
			return nil, err
		}

		if !resp.ShouldBid {
			shouldBid = false
			reasons = append(reasons, fmt.Sprintf("rejected bid: %s", resp.Reason))
		}
	}

	return &bidStrategyResponse{
		bid:    shouldBid,
		reason: strings.Join(reasons, "; "),
	}, nil
}

func (b Bidder) runResourceBidding(ctx context.Context, job *models.Job) (*bidStrategyResponse, error) {
	// parse job resource config
	parsedUsage, err := job.Task().ResourcesConfig.ToResources()
	if err != nil {
		return nil, fmt.Errorf("parsing job resource config: %w", err)
	}
	// calculate resource usage of the job, failure here represents a compute failure.
	resourceUsage, err := b.usageCalculator.Calculate(ctx, *job, *parsedUsage)
	if err != nil {
		return nil, fmt.Errorf("calculating resource usage of job: %w", err)
	}

	// ask the bidding strategy if we should bid on this job
	request := bidstrategy.BidStrategyRequest{
		Job: *job,
	}

	// assume we are bidding unless a request is rejected
	shouldBid := true
	reasons := make([]string, 0, len(b.resourceStrategy))

	// TODO(forrest): this can be parallelized with a wait group, room for improvement here if resource validation
	//  is time consuming.
	for _, s := range b.resourceStrategy {
		strategyType := reflect.TypeOf(s).String()
		resp, err := s.ShouldBidBasedOnUsage(ctx, request, *resourceUsage)

		if err != nil || !resp.ShouldBid {
			log.Ctx(ctx).WithLevel(logger.ErrOrDebug(err)).
				Err(err).
				Str("Job", request.Job.ID).
				Str("Strategy", strategyType).
				Bool("Bid", resp.ShouldBid).
				Str("Reason", resp.Reason).
				Send()
		}

		if err != nil {
			// NB: failure here results in a callback to OnComputeFailure
			return nil, err
		}

		if !resp.ShouldBid {
			shouldBid = false
			reasons = append(reasons, fmt.Sprintf("rejected bid: %s", resp.Reason))
		}
	}

	return &bidStrategyResponse{
		bid:                 shouldBid,
		reason:              strings.Join(reasons, "; "),
		calculatedResources: resourceUsage,
	}, nil
}

// handleBidResult is a helper function to handle the result of the bidding process.
// It updates the execution state based on the result of the bidding process
func (b Bidder) handleBidResult(
	ctx context.Context,
	execution *models.Execution,
	result *bidStrategyResponse,
) {
	var newExecutionValues models.Execution
	var newExecutionState models.ExecutionStateType
	if !result.bid {
		newExecutionState = models.ExecutionStateAskForBidRejected
	} else if execution.DesiredState.StateType == models.ExecutionDesiredStatePending {
		newExecutionState = models.ExecutionStateAskForBidAccepted
	} else {
		newExecutionState = models.ExecutionStateBidAccepted
	}
	newExecutionValues.ComputeState = models.NewExecutionState(newExecutionState).WithMessage(result.reason)

	if result.bid {
		newExecutionValues.AllocateResources(execution.Job.Task().Name, *result.calculatedResources)
	}

	err := b.store.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		NewValues:   newExecutionValues,
		Events:      []models.Event{*models.NewEvent(EventTopicExecutionBidding).WithMessage(result.reason)},
		Condition: store.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{models.ExecutionStateNew},
		},
	})
	// TODO: handle error by either gracefully skipping if the execution is no longer in the created state
	//  or by failing the execution
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update execution state")
		return
	}
}

// handleError is a helper function to handle errors in the bidder.
// It updates the execution state to failed
func (b Bidder) handleError(ctx context.Context, execution *models.Execution, err error) {
	updatedErr := b.store.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateFailed).WithMessage(err.Error()),
		},
		Events: []models.Event{models.EventFromError(EventTopicExecutionBidding, err)},
		Condition: store.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{models.ExecutionStateNew},
		},
	})
	if updatedErr != nil {
		log.Ctx(ctx).Error().Err(updatedErr).Msg("failed to update execution state")
	}
}
