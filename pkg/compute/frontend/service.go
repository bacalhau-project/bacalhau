package frontend

import (
	"context"
	"fmt"
	"strconv"

	"github.com/filecoin-project/bacalhau/pkg/compute/backend"
	"github.com/filecoin-project/bacalhau/pkg/compute/bidstrategy"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type BaseServiceParams struct {
	ID              string
	ExecutionStore  store.ExecutionStore
	UsageCalculator capacity.UsageCalculator
	BidStrategy     bidstrategy.BidStrategy
	Backend         backend.Service
}

// Base implementation of Service
type BaseService struct {
	id              string
	executionStore  store.ExecutionStore
	usageCalculator capacity.UsageCalculator
	bidStrategy     bidstrategy.BidStrategy
	backend         backend.Service
}

func NewBaseService(params BaseServiceParams) BaseService {
	return BaseService{
		id:              params.ID,
		executionStore:  params.ExecutionStore,
		usageCalculator: params.UsageCalculator,
		bidStrategy:     params.BidStrategy,
		backend:         params.Backend,
	}
}

func (s BaseService) GetNodeID() string {
	return s.id
}

func (s BaseService) AskForBid(ctx context.Context, request AskForBidRequest) (AskForBidResponse, error) {
	ctx, span := s.newSpan(ctx, "AskForBid")
	defer span.End()
	log.Ctx(ctx).Debug().Msgf("job created: %s", request.Job.ID)
	jobsReceived.With(prometheus.Labels{"node_id": s.id, "client_id": request.Job.ClientID}).Inc()

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
func (s BaseService) prepareAskForBidShardResponse(
	ctx context.Context,
	request AskForBidRequest,
	shardIndex int,
	shardRequirements model.ResourceUsageData,
	bidStrategyResponse bidstrategy.BidStrategyResponse) (AskForBidShardResponse, error) {
	if !bidStrategyResponse.ShouldBid {
		return AskForBidShardResponse{
			ShardIndex: shardIndex,
			Accepted:   false,
			Reason:     bidStrategyResponse.Reason,
		}, nil
	}

	execution := *store.NewExecution(
		"e-"+uuid.NewString(),
		model.JobShard{Job: &request.Job, Index: shardIndex},
		shardRequirements,
	)

	err := s.executionStore.CreateExecution(ctx, execution)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("error adding shard %s to backlog", execution.Shard)
		return AskForBidShardResponse{
			ShardIndex: shardIndex,
			Accepted:   false,
			Reason:     "error adding shard to backlog",
		}, err
	} else {
		log.Ctx(ctx).Debug().Msgf("bidding for shard %s with execution %s", execution.Shard, execution.ID)
		return AskForBidShardResponse{
			ShardIndex:  shardIndex,
			Accepted:    true,
			ExecutionID: execution.ID,
		}, nil
	}
}

func (s BaseService) BidAccepted(ctx context.Context, request BidAcceptedRequest) (BidAcceptedResult, error) {
	log.Ctx(ctx).Debug().Msgf("bid accepted: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   request.ExecutionID,
		ExpectedState: store.ExecutionStateCreated,
		NewState:      store.ExecutionStateBidAccepted,
	})
	if err != nil {
		return BidAcceptedResult{}, err
	}

	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return BidAcceptedResult{}, err
	}

	// Increment the number of jobs accepted by this compute node:
	jobsAccepted.With(prometheus.Labels{
		"node_id":     s.id,
		"shard_index": strconv.Itoa(execution.Shard.Index),
		"client_id":   execution.Shard.Job.ClientID,
	}).Inc()

	err = s.backend.Run(ctx, execution)
	if err != nil {
		return BidAcceptedResult{}, err
	}
	return BidAcceptedResult{}, nil
}

func (s BaseService) BidRejected(ctx context.Context, request BidRejectedRequest) (BidRejectedResult, error) {
	log.Ctx(ctx).Debug().Msgf("bid rejected: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   request.ExecutionID,
		ExpectedState: store.ExecutionStateCreated,
		NewState:      store.ExecutionStateCancelled,
		Comment:       "bid rejected due to: " + request.Justification,
	})
	if err != nil {
		return BidRejectedResult{}, err
	}
	return BidRejectedResult{}, nil
}

func (s BaseService) ResultAccepted(ctx context.Context, request ResultAcceptedRequest) (ResultAcceptedResult, error) {
	log.Ctx(ctx).Debug().Msgf("results accepted: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   request.ExecutionID,
		ExpectedState: store.ExecutionStateWaitingVerification,
		NewState:      store.ExecutionStateResultAccepted,
	})
	if err != nil {
		return ResultAcceptedResult{}, err
	}
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return ResultAcceptedResult{}, err
	}

	err = s.backend.Publish(ctx, execution)
	if err != nil {
		return ResultAcceptedResult{}, err
	}
	return ResultAcceptedResult{}, nil
}

func (s BaseService) ResultRejected(ctx context.Context, request ResultRejectedRequest) (ResultRejectedResult, error) {
	log.Ctx(ctx).Debug().Msgf("results rejected: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   request.ExecutionID,
		ExpectedState: store.ExecutionStateWaitingVerification,
		NewState:      store.ExecutionStateFailed,
		Comment:       "result rejected due to: " + request.Justification,
	})
	if err != nil {
		return ResultRejectedResult{}, err
	}
	return ResultRejectedResult{}, nil
}

func (s BaseService) CancelJob(ctx context.Context, request CancelJobRequest) (CancelJobResult, error) {
	log.Ctx(ctx).Debug().Msgf("canceling execution: %s", request.ExecutionID)
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return CancelJobResult{}, err
	}
	if execution.State.IsTerminal() {
		return CancelJobResult{}, fmt.Errorf("cannot cancel execution %s in state %s", execution.ID, execution.State)
	}

	if execution.State.IsExecuting() {
		err = s.backend.Cancel(ctx, execution)
		if err != nil {
			return CancelJobResult{}, err
		}
	}

	err = s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID: request.ExecutionID,
		NewState:    store.ExecutionStateCancelled,
		Comment:     "execution canceled due to: " + request.Justification,
	})
	if err != nil {
		return CancelJobResult{}, err
	}
	return CancelJobResult{}, nil
}

func (s BaseService) newSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return system.Span(ctx, "pkg/compute/node", name,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

// Compile-time interface check:
var _ Service = (*BaseService)(nil)
