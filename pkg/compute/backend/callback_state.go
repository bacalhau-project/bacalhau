package backend

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/rs/zerolog/log"
)

type StateUpdateCallbackParams struct {
	ExecutionStore store.ExecutionStore
}

type StateUpdateCallback struct {
	executionStore store.ExecutionStore
}

func NewStateUpdateCallback(params StateUpdateCallbackParams) *StateUpdateCallback {
	return &StateUpdateCallback{
		executionStore: params.ExecutionStore,
	}
}

func (s StateUpdateCallback) OnRunSuccess(ctx context.Context, executionID string, result RunResult) {
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   executionID,
		ExpectedState: store.ExecutionStateRunning,
		NewState:      store.ExecutionStateWaitingVerification,
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("OnRunSuccess: error updating execution %s state to %s: %s",
			executionID, store.ExecutionStateWaitingVerification, err)
	}
}

func (s StateUpdateCallback) OnRunFailure(ctx context.Context, executionID string, runError error) {
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   executionID,
		ExpectedState: store.ExecutionStateRunning,
		NewState:      store.ExecutionStateFailed,
		Comment:       runError.Error(),
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("OnRunFailure: error updating execution %s state to %s: %s",
			executionID, store.ExecutionStateFailed, err)
	}
}

func (s StateUpdateCallback) OnPublishSuccess(ctx context.Context, executionID string, result PublishResult) {
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   executionID,
		ExpectedState: store.ExecutionStatePublishing,
		NewState:      store.ExecutionStateCompleted,
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("OnPublishSuccess: error updating execution %s state to %s: %s",
			executionID, store.ExecutionStateCompleted, err)
	}
}

func (s StateUpdateCallback) OnPublishFailure(ctx context.Context, executionID string, publishError error) {
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID:   executionID,
		ExpectedState: store.ExecutionStatePublishing,
		NewState:      store.ExecutionStateFailed,
		Comment:       publishError.Error(),
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("OnPublishFailure: error updating execution %s state to %s: %s",
			executionID, store.ExecutionStateCompleted, err)
	}
}

func (s StateUpdateCallback) OnCancelSuccess(ctx context.Context, executionID string, result CancelResult) {
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID: executionID,
		NewState:    store.ExecutionStateCancelled,
		Comment:     "Canceled after execution accepted",
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("OnCancelSuccess: error updating execution %s state to %s: %s",
			executionID, store.ExecutionStateCancelled, err)
	}
}

func (s StateUpdateCallback) OnCancelFailure(ctx context.Context, executionID string, cancelError error) {
	err := s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID: executionID,
		NewState:    store.ExecutionStateFailed,
		Comment:     cancelError.Error(),
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("OnCancelFailure: error updating execution %s state to %s: %s",
			executionID, store.ExecutionStateFailed, err)
	}
}

// compile-time interface check
var _ Callback = &StateUpdateCallback{}
