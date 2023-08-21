package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type CallbackStore struct {
	GetExecutionFn         func(ctx context.Context, id string) (store.LocalExecutionState, error)
	GetExecutionsFn        func(ctx context.Context, id string) ([]store.LocalExecutionState, error)
	GetExecutionHistoryFn  func(ctx context.Context, id string) ([]store.LocalStateHistory, error)
	CreateExecutionFn      func(ctx context.Context, execution store.LocalExecutionState) error
	UpdateExecutionStateFn func(ctx context.Context, request store.UpdateExecutionStateRequest) error
	DeleteExecutionFn      func(ctx context.Context, id string) error
	GetExecutionCountFn    func(ctx context.Context, state store.LocalExecutionStateType) (uint64, error)
	CloseFn                func(ctx context.Context) error
}

func (m *CallbackStore) GetExecution(ctx context.Context, id string) (store.LocalExecutionState, error) {
	return m.GetExecutionFn(ctx, id)
}

func (m *CallbackStore) GetExecutions(ctx context.Context, jobID string) ([]store.LocalExecutionState, error) {
	return m.GetExecutionsFn(ctx, jobID)
}

func (m *CallbackStore) GetExecutionHistory(ctx context.Context, id string) ([]store.LocalStateHistory, error) {
	return m.GetExecutionHistoryFn(ctx, id)
}

func (m *CallbackStore) CreateExecution(ctx context.Context, execution store.LocalExecutionState) error {
	return m.CreateExecutionFn(ctx, execution)
}

func (m *CallbackStore) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	return m.UpdateExecutionStateFn(ctx, request)
}

func (m *CallbackStore) DeleteExecution(ctx context.Context, id string) error {
	return m.DeleteExecutionFn(ctx, id)
}

func (m *CallbackStore) GetExecutionCount(ctx context.Context, state store.LocalExecutionStateType) (uint64, error) {
	return m.GetExecutionCountFn(ctx, state)
}

func (m *CallbackStore) Close(ctx context.Context) error {
	return m.CloseFn(ctx)
}
