package compute

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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
	LogServer       logstream.LogStreamServer
}

// Base implementation of Endpoint
type BaseEndpoint struct {
	id              string
	executionStore  store.ExecutionStore
	usageCalculator capacity.UsageCalculator
	bidStrategy     bidstrategy.BidStrategy
	executor        Executor
	logServer       logstream.LogStreamServer
}

func NewBaseEndpoint(params BaseEndpointParams) BaseEndpoint {
	return BaseEndpoint{
		id:              params.ID,
		executionStore:  params.ExecutionStore,
		usageCalculator: params.UsageCalculator,
		bidStrategy:     params.BidStrategy,
		executor:        params.Executor,
		logServer:       params.LogServer,
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
	bidStrategyRequest := bidstrategy.BidStrategyRequest{
		NodeID: s.id,
		Job:    request.Job,
	}

	// Check bidding strategies before having to calculate resource usage
	bidStrategyResponse, err := s.bidStrategy.ShouldBid(ctx, bidStrategyRequest)
	if err != nil {
		return AskForBidResponse{}, fmt.Errorf("error asking bidding strategy if we should bid: %w", err)
	}

	var jobRequirements model.ResourceUsageData
	if bidStrategyResponse.ShouldBid {
		// calculate resource requirements for this job
		jobRequirements, err = s.usageCalculator.Calculate(
			ctx, request.Job, capacity.ParseResourceUsageConfig(request.Job.Spec.Resources))
		if err != nil {
			return AskForBidResponse{}, fmt.Errorf("error calculating job requirements: %w", err)
		}

		// Check bidding strategies after calculating resource usage
		bidStrategyResponse, err = s.bidStrategy.ShouldBidBasedOnUsage(ctx, bidStrategyRequest, jobRequirements)
		if err != nil {
			return AskForBidResponse{}, fmt.Errorf("error asking bidding strategy if we should bid: %w", err)
		}
	}

	return s.prepareAskForBidResponse(ctx, request, jobRequirements, bidStrategyResponse)
}

// Enqueues the job in the execution executionStore, and returns the response.
// Failure to enqueue the job will return BOTH an error and a response with Accepted=false.
func (s BaseEndpoint) prepareAskForBidResponse(
	ctx context.Context,
	request AskForBidRequest,
	resourceUsage model.ResourceUsageData,
	bidStrategyResponse bidstrategy.BidStrategyResponse) (AskForBidResponse, error) {
	if !bidStrategyResponse.ShouldBid {
		return AskForBidResponse{
			ExecutionMetadata: ExecutionMetadata{
				JobID: request.Job.Metadata.ID,
			},
			Accepted: false,
			Reason:   bidStrategyResponse.Reason,
		}, nil
	}

	execution := *store.NewExecution(
		"e-"+uuid.NewString(),
		request.Job,
		request.SourcePeerID,
		resourceUsage,
	)

	err := s.executionStore.CreateExecution(ctx, execution)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("error adding job %s to backlog", execution.Job)
		return AskForBidResponse{
			ExecutionMetadata: ExecutionMetadata{
				JobID: request.Job.Metadata.ID,
			},
			Accepted: false,
			Reason:   "error adding job to backlog",
		}, err
	} else {
		log.Ctx(ctx).Debug().Msgf("bidding for job %s with execution %s", execution.Job, execution.ID)
		return AskForBidResponse{
			ExecutionMetadata: ExecutionMetadata{
				ExecutionID: execution.ID,
				JobID:       request.Job.Metadata.ID,
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

func (s BaseEndpoint) ExecutionLogs(ctx context.Context, request ExecutionLogsRequest) (ExecutionLogsResponse, error) {
	log.Ctx(ctx).Debug().Msgf("processing log request for %s", request.ExecutionID)
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return ExecutionLogsResponse{}, err
	}

	return ExecutionLogsResponse{
		Address:           s.logServer.Address,
		ExecutionFinished: execution.State.IsTerminal(),
	}, nil
}

// Compile-time interface check:
var _ Endpoint = (*BaseEndpoint)(nil)
