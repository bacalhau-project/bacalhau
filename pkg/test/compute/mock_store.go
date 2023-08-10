package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type CallbackStore struct {
	GetExecutionFn         func(ctx context.Context, id string) (store.Execution, error)
	GetExecutionsFn        func(ctx context.Context, id string) ([]store.Execution, error)
	GetLiveExecutionsFn    func(ctx context.Context) ([]store.Execution, error)
	GetExecutionHistoryFn  func(ctx context.Context, id string) ([]store.ExecutionHistory, error)
	CreateExecutionFn      func(ctx context.Context, execution store.Execution) error
	UpdateExecutionStateFn func(ctx context.Context, request store.UpdateExecutionStateRequest) error
	DeleteExecutionFn      func(ctx context.Context, id string) error
	GetExecutionCountFn    func(ctx context.Context, state store.ExecutionState) (uint64, error)
	CloseFn                func(ctx context.Context) error
}

func (m *CallbackStore) GetExecution(ctx context.Context, id string) (store.Execution, error) {
	return m.GetExecutionFn(ctx, id)
}

func (m *CallbackStore) GetExecutions(ctx context.Context, jobID string) ([]store.Execution, error) {
	return m.GetExecutionsFn(ctx, jobID)
}

func (m *CallbackStore) GetLiveExecutions(ctx context.Context) ([]store.Execution, error) {
	return m.GetLiveExecutionsFn(ctx)
}

func (m *CallbackStore) GetExecutionHistory(ctx context.Context, id string) ([]store.ExecutionHistory, error) {
	return m.GetExecutionHistoryFn(ctx, id)
}

func (m *CallbackStore) CreateExecution(ctx context.Context, execution store.Execution) error {
	return m.CreateExecutionFn(ctx, execution)
}

func (m *CallbackStore) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	return m.UpdateExecutionStateFn(ctx, request)
}

func (m *CallbackStore) DeleteExecution(ctx context.Context, id string) error {
	return m.DeleteExecutionFn(ctx, id)
}

func (m *CallbackStore) GetExecutionCount(ctx context.Context, state store.ExecutionState) (uint64, error) {
	return m.GetExecutionCountFn(ctx, state)
}

func (m *CallbackStore) Close(ctx context.Context) error {
	return m.CloseFn(ctx)
}
