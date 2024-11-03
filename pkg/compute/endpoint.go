package compute

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

type BaseEndpointParams struct {
	ID             string
	ExecutionStore store.ExecutionStore
	LogServer      logstream.Server
}

// Base implementation of Endpoint
type BaseEndpoint struct {
	id             string
	executionStore store.ExecutionStore
	logServer      logstream.Server
}

func NewBaseEndpoint(params BaseEndpointParams) BaseEndpoint {
	return BaseEndpoint{
		id:             params.ID,
		executionStore: params.ExecutionStore,
		logServer:      params.LogServer,
	}
}

func (s BaseEndpoint) GetNodeID() string {
	return s.id
}

func (s BaseEndpoint) AskForBid(
	ctx context.Context, request messages.AskForBidRequest) (messages.AskForBidResponse, error) {
	log.Ctx(ctx).Debug().Msgf("asked to bid on: %+v", request)
	jobsReceived.Add(ctx, 1)

	if !request.WaitForApproval {
		request.Execution.DesiredState.StateType = models.ExecutionDesiredStateRunning
	}

	// Create the execution in the store. The bidder will asynchronously handle the bid request.
	if err := s.executionStore.CreateExecution(ctx, *request.Execution); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error creating execution")
		return messages.AskForBidResponse{ExecutionMetadata: messages.ExecutionMetadata{
			ExecutionID: request.Execution.ID,
			JobID:       request.Execution.JobID,
		}}, err
	}

	return messages.AskForBidResponse{ExecutionMetadata: messages.ExecutionMetadata{
		ExecutionID: request.Execution.ID,
		JobID:       request.Execution.JobID,
	}}, nil
}

func (s BaseEndpoint) BidAccepted(
	ctx context.Context, request messages.BidAcceptedRequest) (messages.BidAcceptedResponse, error) {
	log.Ctx(ctx).Debug().Msgf("bid accepted: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: request.ExecutionID,
		Condition: store.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{
				models.ExecutionStateNew, models.ExecutionStateAskForBidAccepted},
		},
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateBidAccepted),
		},
	})
	if err != nil {
		return messages.BidAcceptedResponse{}, err
	}

	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return messages.BidAcceptedResponse{}, err
	}

	return messages.BidAcceptedResponse{
		ExecutionMetadata: messages.NewExecutionMetadata(execution),
	}, nil
}

func (s BaseEndpoint) BidRejected(
	ctx context.Context, request messages.BidRejectedRequest) (messages.BidRejectedResponse, error) {
	log.Ctx(ctx).Debug().Msgf("bid rejected: %s", request.ExecutionID)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: request.ExecutionID,
		Condition: store.UpdateExecutionCondition{
			ExpectedStates: []models.ExecutionStateType{
				models.ExecutionStateNew, models.ExecutionStateAskForBidAccepted},
		},
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateBidRejected).WithMessage(request.Justification),
		},
	})
	if err != nil {
		return messages.BidRejectedResponse{}, err
	}
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return messages.BidRejectedResponse{}, err
	}
	return messages.BidRejectedResponse{
		ExecutionMetadata: messages.NewExecutionMetadata(execution),
	}, nil
}

func (s BaseEndpoint) CancelExecution(
	ctx context.Context, request messages.CancelExecutionRequest) (messages.CancelExecutionResponse, error) {
	log.Ctx(ctx).Debug().Msgf("canceling execution %s due to %s", request.ExecutionID, request.Justification)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: request.ExecutionID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateCancelled).WithMessage(request.Justification),
		},
	})
	if err != nil {
		return messages.CancelExecutionResponse{}, err
	}
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return messages.CancelExecutionResponse{}, err
	}
	return messages.CancelExecutionResponse{
		ExecutionMetadata: messages.NewExecutionMetadata(execution),
	}, nil
}

func (s BaseEndpoint) ExecutionLogs(ctx context.Context, request messages.ExecutionLogsRequest) (
	<-chan *concurrency.AsyncResult[models.ExecutionLog], error,
) {
	return s.logServer.GetLogStream(ctx, request)
}

// Compile-time interface check:
var _ Endpoint = (*BaseEndpoint)(nil)
