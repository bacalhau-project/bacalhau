package compute

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type BaseEndpointParams struct {
	ID              string
	ExecutionStore  store.ExecutionStore
	UsageCalculator capacity.UsageCalculator
	Bidder          Bidder
	Executor        Executor
	LogServer       logstream.LogStreamServer
}

// Base implementation of Endpoint
type BaseEndpoint struct {
	id              string
	executionStore  store.ExecutionStore
	usageCalculator capacity.UsageCalculator
	bidder          Bidder
	executor        Executor
	logServer       logstream.LogStreamServer
}

func NewBaseEndpoint(params BaseEndpointParams) BaseEndpoint {
	return BaseEndpoint{
		id:              params.ID,
		executionStore:  params.ExecutionStore,
		usageCalculator: params.UsageCalculator,
		bidder:          params.Bidder,
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

	go s.bidder.RunBidding(ctx, request, s.usageCalculator) // TODO: context shareable?

	return AskForBidResponse{ExecutionMetadata: ExecutionMetadata{
		ExecutionID: request.ExecutionID,
		JobID:       request.JobID,
	}}, nil
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
