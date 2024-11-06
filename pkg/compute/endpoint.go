package compute

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

type BaseEndpointParams struct {
	ExecutionStore store.ExecutionStore
}

// Base implementation of Endpoint
type BaseEndpoint struct {
	executionStore store.ExecutionStore
}

func NewBaseEndpoint(params BaseEndpointParams) BaseEndpoint {
	return BaseEndpoint{
		executionStore: params.ExecutionStore,
	}
}

func (s BaseEndpoint) AskForBid(
	ctx context.Context, request legacy.AskForBidRequest) (legacy.AskForBidResponse, error) {
	log.Ctx(ctx).Debug().Msgf("asked to bid on: %+v", request)
	jobsReceived.Add(ctx, 1)

	// Create the execution in the store. The bidder will asynchronously handle the bid request.
	if err := s.executionStore.CreateExecution(ctx, *request.Execution); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error creating execution")
		return legacy.AskForBidResponse{ExecutionMetadata: legacy.ExecutionMetadata{
			ExecutionID: request.Execution.ID,
			JobID:       request.Execution.JobID,
		}}, err
	}

	return legacy.AskForBidResponse{ExecutionMetadata: legacy.ExecutionMetadata{
		ExecutionID: request.Execution.ID,
		JobID:       request.Execution.JobID,
	}}, nil
}

func (s BaseEndpoint) BidAccepted(
	ctx context.Context, request legacy.BidAcceptedRequest) (legacy.BidAcceptedResponse, error) {
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
		return legacy.BidAcceptedResponse{}, err
	}

	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return legacy.BidAcceptedResponse{}, err
	}

	return legacy.BidAcceptedResponse{
		ExecutionMetadata: legacy.NewExecutionMetadata(execution),
	}, nil
}

func (s BaseEndpoint) BidRejected(
	ctx context.Context, request legacy.BidRejectedRequest) (legacy.BidRejectedResponse, error) {
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
		return legacy.BidRejectedResponse{}, err
	}
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return legacy.BidRejectedResponse{}, err
	}
	return legacy.BidRejectedResponse{
		ExecutionMetadata: legacy.NewExecutionMetadata(execution),
	}, nil
}

func (s BaseEndpoint) CancelExecution(
	ctx context.Context, request legacy.CancelExecutionRequest) (legacy.CancelExecutionResponse, error) {
	log.Ctx(ctx).Debug().Msgf("canceling execution %s due to %s", request.ExecutionID, request.Justification)
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
		ExecutionID: request.ExecutionID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateCancelled).WithMessage(request.Justification),
		},
	})
	if err != nil {
		return legacy.CancelExecutionResponse{}, err
	}
	execution, err := s.executionStore.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return legacy.CancelExecutionResponse{}, err
	}
	return legacy.CancelExecutionResponse{
		ExecutionMetadata: legacy.NewExecutionMetadata(execution),
	}, nil
}

// Compile-time interface check:
var _ Endpoint = (*BaseEndpoint)(nil)
