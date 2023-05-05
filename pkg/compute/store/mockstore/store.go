package mockstore

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type MockExecutionStore struct {
	mock.Mock
}

func (m *MockExecutionStore) GetExecution(ctx context.Context, id string) (store.Execution, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(store.Execution), args.Error(1)
}

func (m *MockExecutionStore) GetExecutions(ctx context.Context, jobID string) ([]store.Execution, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).([]store.Execution), args.Error(1)
}

func (m *MockExecutionStore) GetExecutionHistory(ctx context.Context, id string) ([]store.ExecutionHistory, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]store.ExecutionHistory), args.Error(1)
}

func (m *MockExecutionStore) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

func (m *MockExecutionStore) DeleteExecution(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockExecutionStore) GetExecutionCount(ctx context.Context) (uint, error) {
	args := m.Called(ctx)
	return uint(args.Int(0)), args.Error(1)
}

func (m *MockExecutionStore) CreateExecution(ctx context.Context, execution store.Execution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}
