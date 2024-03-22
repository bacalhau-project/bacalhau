package compute

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"

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
	LogServer       *logstream.Server
}

// Base implementation of Endpoint
type BaseEndpoint struct {
	id              string
	executionStore  store.ExecutionStore
	usageCalculator capacity.UsageCalculator
	bidder          Bidder
	executor        Executor
	logServer       *logstream.Server
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

	// parse job resource config
	parsedUsage, err := request.Execution.Job.Task().ResourcesConfig.ToResources()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error parsing job resource config")
		return AskForBidResponse{ExecutionMetadata: ExecutionMetadata{
			ExecutionID: request.Execution.ID,
			JobID:       request.Execution.JobID,
		}}, err
	}

	// calculate resource usage of the job
	resourceUsage, err := s.usageCalculator.Calculate(ctx, *request.Execution.Job, *parsedUsage)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error calculating resource requirements for job")
		return AskForBidResponse{ExecutionMetadata: ExecutionMetadata{
			ExecutionID: request.Execution.ID,
			JobID:       request.Execution.JobID,
		}}, err
	}
	// update the execution with the calculated resource usage
	request.Execution.AllocateResources(request.Execution.Job.Task().Name, *resourceUsage)

	// TODO: context shareable?
	// TODO: lots of things are happening here
	go func() {
		routingMetadata := RoutingMetadata{
			// the source of this response is the bidders nodeID.
			SourcePeerID: s.id,
			// the target of this response is the source of the request.
			TargetPeerID: request.SourcePeerID,
		}
		executionMetadata := ExecutionMetadata{
			ExecutionID: request.Execution.ID,
			JobID:       request.Execution.JobID,
		}
		result, err := s.bidder.RunBidding(ctx, request, resourceUsage)
		if err != nil {
			// TODO move the call back to the base endpoint, off the bidder
			s.bidder.callback.OnComputeFailure(ctx, ComputeError{
				RoutingMetadata:   routingMetadata,
				ExecutionMetadata: executionMetadata,
				Err:               err.Error(),
			})
		}
		if !request.WaitForApproval {
			if !result.Accepted || result.Wait {
				s.bidder.callback.OnComputeFailure(ctx, ComputeError{
					RoutingMetadata:   routingMetadata,
					ExecutionMetadata: executionMetadata,
					Err:               fmt.Sprintf("Job rejected: %s", result.Reason),
				})
				return
			}

			// TODO failure in either of these cases represents a compute failure
			execution := store.NewLocalExecutionState(request.Execution, request.SourcePeerID)
			execution.State = store.ExecutionStateBidAccepted
			if err := s.executionStore.CreateExecution(ctx, *execution); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("Unable to create execution state")
				s.bidder.callback.OnComputeFailure(ctx, ComputeError{
					RoutingMetadata:   routingMetadata,
					ExecutionMetadata: executionMetadata,
					Err:               fmt.Sprintf("Unable to create execution state: %s", err),
				})
				return
			}
			err := s.executor.Run(ctx, *execution)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("Unable to run execution")
				s.bidder.callback.OnComputeFailure(ctx, ComputeError{
					RoutingMetadata:   routingMetadata,
					ExecutionMetadata: executionMetadata,
					Err:               fmt.Sprintf("Unable to run execution: %s", err),
				})
			}
			return
		}

		// TODO another compute failure here
		// if we are bidding or waiting create an execution
		if result.Wait || result.Accepted {
			execution := store.NewLocalExecutionState(request.Execution, request.SourcePeerID)
			if err := s.executionStore.CreateExecution(ctx, *execution); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("Unable to create execution state")
				s.bidder.callback.OnComputeFailure(ctx, ComputeError{
					RoutingMetadata:   routingMetadata,
					ExecutionMetadata: executionMetadata,
					Err:               fmt.Sprintf("Unable to create execution state: %s", err),
				})
				return
			}
		}

		// if we are not bidding and not wait return a response, we can't do this job. mark as complete then bail
		if !result.Accepted && !result.Wait {
			s.bidder.callback.OnBidComplete(ctx, *result)
			return
		}

		// were not waiting return a response.
		if !result.Wait {
			s.bidder.callback.OnBidComplete(ctx, *result)
		}
	}()

	return AskForBidResponse{ExecutionMetadata: ExecutionMetadata{
		ExecutionID: request.Execution.ID,
		JobID:       request.Execution.JobID,
	}}, nil
}

func (s BaseEndpoint) BidAccepted(ctx context.Context, request BidAcceptedRequest) (BidAcceptedResponse, error) {
	log.Ctx(ctx).Debug().Msgf("bid accepted: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:    request.ExecutionID,
		ExpectedStates: []store.LocalExecutionStateType{store.ExecutionStateCreated},
		NewState:       store.ExecutionStateBidAccepted,
	})
	if err != nil {
		return BidAcceptedResponse{}, err
	}

	localExecutionState, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return BidAcceptedResponse{}, err
	}

	// Increment the number of jobs accepted by this compute node:
	jobsAccepted.Add(ctx, 1)

	err = s.executor.Run(ctx, localExecutionState)
	if err != nil {
		return BidAcceptedResponse{}, err
	}
	return BidAcceptedResponse{
		ExecutionMetadata: NewExecutionMetadata(localExecutionState.Execution),
	}, nil
}

func (s BaseEndpoint) BidRejected(ctx context.Context, request BidRejectedRequest) (BidRejectedResponse, error) {
	log.Ctx(ctx).Debug().Msgf("bid rejected: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:    request.ExecutionID,
		ExpectedStates: []store.LocalExecutionStateType{store.ExecutionStateCreated},
		NewState:       store.ExecutionStateCancelled,
		Comment:        "bid rejected due to: " + request.Justification,
	})
	if err != nil {
		return BidRejectedResponse{}, err
	}
	localExecutionState, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return BidRejectedResponse{}, err
	}
	return BidRejectedResponse{
		ExecutionMetadata: NewExecutionMetadata(localExecutionState.Execution),
	}, nil
}

func (s BaseEndpoint) CancelExecution(ctx context.Context, request CancelExecutionRequest) (CancelExecutionResponse, error) {
	log.Ctx(ctx).Debug().Msgf("canceling execution %s due to %s", request.ExecutionID, request.Justification)
	localExecutionState, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return CancelExecutionResponse{}, err
	}
	if localExecutionState.State.IsTerminal() {
		return CancelExecutionResponse{}, fmt.Errorf("cannot cancel execution %s in state %s",
			localExecutionState.Execution.ID, localExecutionState.State)
	}

	if localExecutionState.State.IsExecuting() {
		err = s.executor.Cancel(ctx, localExecutionState)
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
		ExecutionMetadata: NewExecutionMetadata(localExecutionState.Execution),
	}, nil
}

func (s BaseEndpoint) ExecutionLogs(ctx context.Context, request ExecutionLogsRequest) (
	<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	return s.logServer.GetLogStream(ctx, executor.LogStreamRequest{
		ExecutionID: request.ExecutionID,
		Tail:        request.Tail,
		Follow:      request.Follow,
	})
}

// Compile-time interface check:
var _ Endpoint = (*BaseEndpoint)(nil)
