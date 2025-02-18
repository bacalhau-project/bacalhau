package compute

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
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

func (b Bidder) RunBidding(ctx context.Context, execution *models.Execution) error {
	bidResult, err := b.doBidding(ctx, execution)
	if err != nil {
		return b.handleError(ctx, execution, err)
	}
	b.handleBidResult(ctx, execution, bidResult)
	return nil
}

// doBidding returns a response based on the below semantics. We will only bid if both semantic
// and resource strategies approve the bid. If semantic bidding rejects, we short-circuit and
// don't evaluate resource strategies.
// | SemanticShouldBid | ResourceShouldBid | ShouldBid |
// |-------------------|-------------------|------------|
// |      true         |      true         |   true     |
// |      true         |      false        |   false    |
// |      false        |       N/A         |   false    |
func (b Bidder) doBidding(ctx context.Context, execution *models.Execution) (*bidStrategyResponse, error) {
	// NB(forrest): always run semantic bidding before resource bidding since generally there isn't much point in
	// calling resource strategies that require DiskUsageCalculator.Calculate (a precursor to checking bidding) if
	// semantically the job cannot run.
	semanticResponse, err := b.runSemanticBidding(ctx, execution)
	if err != nil {
		return nil, err
	}

	// we shouldn't bid, bail.
	if !semanticResponse.bid {
		return semanticResponse, nil
	}

	// else the request is semantically biddable, calculate resource usage and check resource-based bidding.
	resourceResponse, err := b.runResourceBidding(ctx, execution)
	if err != nil {
		return nil, err
	}

	return resourceResponse, nil
}

func (b Bidder) runSemanticBidding(ctx context.Context, execution *models.Execution) (*bidStrategyResponse, error) {
	// ask the bidding strategy if we should bid on this job
	request := bidstrategy.BidStrategyRequest{
		Job: *execution.Job,
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

func (b Bidder) runResourceBidding(ctx context.Context, execution *models.Execution) (*bidStrategyResponse, error) {
	// parse job resource config
	parsedUsage, err := execution.Job.Task().ResourcesConfig.ToResources()
	if err != nil {
		return nil, fmt.Errorf("parsing job resource config: %w", err)
	}
	// calculate resource usage of the job, failure here represents a compute failure.
	resourceUsage, err := b.usageCalculator.Calculate(ctx, execution, *parsedUsage)
	if err != nil {
		return nil, fmt.Errorf("calculating resource usage of job: %w", err)
	}

	// ask the bidding strategy if we should bid on this job
	request := bidstrategy.BidStrategyRequest{
		Job: *execution.Job,
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
) error {
	var newExecutionValues models.Execution
	var newExecutionState models.ExecutionStateType
	var events []*models.Event
	if !result.bid {
		newExecutionState = models.ExecutionStateAskForBidRejected
	} else if execution.DesiredState.StateType == models.ExecutionDesiredStatePending {
		newExecutionState = models.ExecutionStateAskForBidAccepted
	} else {
		newExecutionState = models.ExecutionStateBidAccepted
	}
	newExecutionValues.ComputeState = models.NewExecutionState(newExecutionState).WithMessage(result.reason)

	if result.bid {
		if result.calculatedResources != nil {
			newExecutionValues.AllocateResources(execution.Job.Task().Name, *result.calculatedResources)
		} else {
			log.Ctx(ctx).Error().Msg("calculatedResources is nil despite bid being true")
		}
	} else {
		// only add an event if the bid was rejected with more information
		events = append(events, models.NewEvent(EventTopicExecutionScanning).WithMessage(result.reason))
	}

	err := b.store.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		NewValues:   newExecutionValues,
		Events:      events,
		Condition: store.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{models.ExecutionStateNew},
		},
	})

	if err != nil {
		var invalidStateErr store.ErrInvalidExecutionState
		if errors.As(err, &invalidStateErr) {
			log.Ctx(ctx).Debug().
				Err(err).
				Str("executionID", execution.ID).
				Str("expectedState", models.ExecutionStateNew.String()).
				Str("actualState", invalidStateErr.Actual.String()).
				Msg("skipping execution state update - execution no longer in expected state")
			return nil
		}

		// Propagate the error to be handled by the execution watcher
		return bacerrors.Wrap(err, "failed to update execution state for execution %s", execution.ID)
	}
	return nil
}

// handleError is a helper function to handle errors in the bidder.
// It updates the execution state to failed
func (b Bidder) handleError(ctx context.Context, execution *models.Execution, err error) error {
	updatedErr := b.store.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateFailed).WithMessage(err.Error()),
		},
		Events: []*models.Event{models.EventFromError(EventTopicExecutionScanning, err)},
		Condition: store.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{models.ExecutionStateNew},
		},
	})
	if updatedErr != nil {
		log.Ctx(ctx).Error().Err(updatedErr).Msg("failed to update execution state")
		return updatedErr
	}
	return nil
}
