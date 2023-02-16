package compute

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/compute/bidstrategy"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type BaseEndpointParams struct {
	ID              string
	ExecutionStore  store.ExecutionStore
	UsageCalculator capacity.UsageCalculator
	BidStrategy     bidstrategy.BidStrategy
	Executor        Executor
}

// Base implementation of Endpoint
type BaseEndpoint struct {
	id              string
	executionStore  store.ExecutionStore
	usageCalculator capacity.UsageCalculator
	bidStrategy     bidstrategy.BidStrategy
	executor        Executor
}

func NewBaseEndpoint(params BaseEndpointParams) BaseEndpoint {
	return BaseEndpoint{
		id:              params.ID,
		executionStore:  params.ExecutionStore,
		usageCalculator: params.UsageCalculator,
		bidStrategy:     params.BidStrategy,
		executor:        params.Executor,
	}
}

func (s BaseEndpoint) GetNodeID() string {
	return s.id
}

func (s BaseEndpoint) AskForBid(ctx context.Context, request AskForBidRequest) (AskForBidResponse, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/compute.BaseEndpoint.AskForBid", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()
	log.Ctx(ctx).Debug().Msgf("asked to bid on: %+v", request)
	jobsReceived.Add(ctx, 1)

	// ask the bidding strategy if we should bid on this job
	// TODO: we should check at the shard level, not the job level
	bidStrategyRequest := bidstrategy.BidStrategyRequest{
		NodeID: s.id,
		Job:    request.Job,
	}

	// Check bidding strategies before having to calculate resource usage
	bidStrategyResponse, err := s.bidStrategy.ShouldBid(ctx, bidStrategyRequest)
	if err != nil {
		return AskForBidResponse{}, fmt.Errorf("error asking bidding strategy if we should bid: %w", err)
	}

	var shardRequirements model.ResourceUsageData
	if bidStrategyResponse.ShouldBid {
		// calculate resource requirements for this job
		shardRequirements, err = s.usageCalculator.Calculate(
			ctx, request.Job, capacity.ParseResourceUsageConfig(request.Job.Spec.Resources))
		if err != nil {
			return AskForBidResponse{}, fmt.Errorf("error calculating job requirements: %w", err)
		}

		// Check bidding strategies after calculating resource usage
		bidStrategyResponse, err = s.bidStrategy.ShouldBidBasedOnUsage(ctx, bidStrategyRequest, shardRequirements)
		if err != nil {
			return AskForBidResponse{}, fmt.Errorf("error asking bidding strategy if we should bid: %w", err)
		}
	}

	// prepare the response, which can include partial bids
	var shardResponses []AskForBidShardResponse
	var enqueueErr error
	var acceptedShards = 0
	for _, shardIndex := range request.ShardIndexes {
		var shardResponse AskForBidShardResponse
		shardResponse, enqueueErr = s.prepareAskForBidShardResponse(ctx, request, shardIndex, shardRequirements, bidStrategyResponse)
		shardResponses = append(shardResponses, shardResponse)
		if shardResponse.Accepted {
			acceptedShards++
		}
	}

	// if we didn't accept any shard, and an error occurred, return it instead of shard level response.
	if enqueueErr != nil && acceptedShards == 0 {
		return AskForBidResponse{}, fmt.Errorf("error preparing shard responses: %w", enqueueErr)
	}

	return AskForBidResponse{ShardResponse: shardResponses}, nil
}

// Enqueues the shard in the execution executionStore, and returns the shard response.
// Failure to enqueue the shard will return BOTH an error and a shard response with Accepted=false.
func (s BaseEndpoint) prepareAskForBidShardResponse(
	ctx context.Context,
	request AskForBidRequest,
	shardIndex int,
	shardRequirements model.ResourceUsageData,
	bidStrategyResponse bidstrategy.BidStrategyResponse) (AskForBidShardResponse, error) {
	if !bidStrategyResponse.ShouldBid {
		return AskForBidShardResponse{
			ExecutionMetadata: ExecutionMetadata{
				JobID:      request.Job.Metadata.ID,
				ShardIndex: shardIndex,
			},
			Accepted: false,
			Reason:   bidStrategyResponse.Reason,
		}, nil
	}

	execution := *store.NewExecution(
		"e-"+uuid.NewString(),
		model.JobShard{Job: &request.Job, Index: shardIndex},
		request.SourcePeerID,
		shardRequirements,
	)

	err := s.executionStore.CreateExecution(ctx, execution)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("error adding shard %s to backlog", execution.Shard)
		return AskForBidShardResponse{
			ExecutionMetadata: ExecutionMetadata{
				JobID:      request.Job.Metadata.ID,
				ShardIndex: shardIndex,
			},
			Accepted: false,
			Reason:   "error adding shard to backlog",
		}, err
	} else {
		log.Ctx(ctx).Debug().Msgf("bidding for shard %s with execution %s", execution.Shard, execution.ID)
		return AskForBidShardResponse{
			ExecutionMetadata: ExecutionMetadata{
				ExecutionID: execution.ID,
				JobID:       request.Job.Metadata.ID,
				ShardIndex:  shardIndex,
			},
			Accepted: true,
		}, nil
	}
}

func (s BaseEndpoint) BidAccepted(ctx context.Context, request BidAcceptedRequest) (BidAcceptedResponse, error) {
	log.Ctx(ctx).Debug().Msgf("bid accepted: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   request.ExecutionID,
		ExpectedState: store.ExecutionStateCreated,
		NewState:      store.ExecutionStateBidAccepted,
	})
	if err != nil {
		return BidAcceptedResponse{}, err
	}

	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return BidAcceptedResponse{}, err
	}

	// Increment the number of jobs accepted by this compute node:
	jobsAccepted.Add(ctx, 1)

	err = s.executor.Run(ctx, execution)
	if err != nil {
		return BidAcceptedResponse{}, err
	}
	return BidAcceptedResponse{
		ExecutionMetadata: NewExecutionMetadata(execution),
	}, nil
}

func (s BaseEndpoint) BidRejected(ctx context.Context, request BidRejectedRequest) (BidRejectedResponse, error) {
	log.Ctx(ctx).Debug().Msgf("bid rejected: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   request.ExecutionID,
		ExpectedState: store.ExecutionStateCreated,
		NewState:      store.ExecutionStateCancelled,
		Comment:       "bid rejected due to: " + request.Justification,
	})
	if err != nil {
		return BidRejectedResponse{}, err
	}
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return BidRejectedResponse{}, err
	}
	return BidRejectedResponse{
		ExecutionMetadata: NewExecutionMetadata(execution),
	}, nil
}

func (s BaseEndpoint) ResultAccepted(ctx context.Context, request ResultAcceptedRequest) (ResultAcceptedResponse, error) {
	log.Ctx(ctx).Debug().Msgf("results accepted: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   request.ExecutionID,
		ExpectedState: store.ExecutionStateWaitingVerification,
		NewState:      store.ExecutionStateResultAccepted,
	})
	if err != nil {
		return ResultAcceptedResponse{}, err
	}
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return ResultAcceptedResponse{}, err
	}

	err = s.executor.Publish(ctx, execution)
	if err != nil {
		return ResultAcceptedResponse{}, err
	}
	return ResultAcceptedResponse{
		ExecutionMetadata: NewExecutionMetadata(execution),
	}, nil
}

func (s BaseEndpoint) ResultRejected(ctx context.Context, request ResultRejectedRequest) (ResultRejectedResponse, error) {
	log.Ctx(ctx).Debug().Msgf("results rejected: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   request.ExecutionID,
		ExpectedState: store.ExecutionStateWaitingVerification,
		NewState:      store.ExecutionStateFailed,
		Comment:       "result rejected due to: " + request.Justification,
	})
	if err != nil {
		return ResultRejectedResponse{}, err
	}
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return ResultRejectedResponse{}, err
	}
	return ResultRejectedResponse{
		ExecutionMetadata: NewExecutionMetadata(execution),
	}, nil
}

func (s BaseEndpoint) CancelExecution(ctx context.Context, request CancelExecutionRequest) (CancelExecutionResponse, error) {
	log.Ctx(ctx).Debug().Msgf("canceling execution %s due to %s", request.ExecutionID, request.Justification)
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return CancelExecutionResponse{}, err
	}
	if execution.State.IsTerminal() {
		return CancelExecutionResponse{}, fmt.Errorf("cannot cancel execution %s in state %s", execution.ID, execution.State)
	}

	if execution.State.IsExecuting() {
		err = s.executor.Cancel(ctx, execution)
		if err != nil {
			return CancelExecutionResponse{}, err
		}
	}

	err = s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID: request.ExecutionID,
		NewState:    store.ExecutionStateCancelled,
		Comment:     "execution canceled due to: " + request.Justification,
	})
	if err != nil {
		return CancelExecutionResponse{}, err
	}
	return CancelExecutionResponse{
		ExecutionMetadata: NewExecutionMetadata(execution),
	}, nil
}

// Compile-time interface check:
var _ Endpoint = (*BaseEndpoint)(nil)
