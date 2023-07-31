// Code generated by MockGen. DO NOT EDIT.
// Source: types.go

// Package store is a generated GoMock package.
package store

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockExecutionStore is a mock of ExecutionStore interface.
type MockExecutionStore struct {
	ctrl     *gomock.Controller
	recorder *MockExecutionStoreMockRecorder
}

// MockExecutionStoreMockRecorder is the mock recorder for MockExecutionStore.
type MockExecutionStoreMockRecorder struct {
	mock *MockExecutionStore
}

// NewMockExecutionStore creates a new mock instance.
func NewMockExecutionStore(ctrl *gomock.Controller) *MockExecutionStore {
	mock := &MockExecutionStore{ctrl: ctrl}
	mock.recorder = &MockExecutionStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExecutionStore) EXPECT() *MockExecutionStoreMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockExecutionStore) Close(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockExecutionStoreMockRecorder) Close(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockExecutionStore)(nil).Close), ctx)
}

// CreateExecution mocks base method.
func (m *MockExecutionStore) CreateExecution(ctx context.Context, execution Execution) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateExecution", ctx, execution)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateExecution indicates an expected call of CreateExecution.
func (mr *MockExecutionStoreMockRecorder) CreateExecution(ctx, execution interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateExecution", reflect.TypeOf((*MockExecutionStore)(nil).CreateExecution), ctx, execution)
}

// DeleteExecution mocks base method.
func (m *MockExecutionStore) DeleteExecution(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteExecution", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteExecution indicates an expected call of DeleteExecution.
func (mr *MockExecutionStoreMockRecorder) DeleteExecution(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteExecution", reflect.TypeOf((*MockExecutionStore)(nil).DeleteExecution), ctx, id)
}

// GetExecution mocks base method.
func (m *MockExecutionStore) GetExecution(ctx context.Context, id string) (Execution, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExecution", ctx, id)
	ret0, _ := ret[0].(Execution)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExecution indicates an expected call of GetExecution.
func (mr *MockExecutionStoreMockRecorder) GetExecution(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExecution", reflect.TypeOf((*MockExecutionStore)(nil).GetExecution), ctx, id)
}

// GetExecutionCount mocks base method.
func (m *MockExecutionStore) GetExecutionCount(ctx context.Context, state ExecutionState) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExecutionCount", ctx, state)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExecutionCount indicates an expected call of GetExecutionCount.
func (mr *MockExecutionStoreMockRecorder) GetExecutionCount(ctx, state interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExecutionCount", reflect.TypeOf((*MockExecutionStore)(nil).GetExecutionCount), ctx, state)
}

// GetExecutionHistory mocks base method.
func (m *MockExecutionStore) GetExecutionHistory(ctx context.Context, id string) ([]ExecutionHistory, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExecutionHistory", ctx, id)
	ret0, _ := ret[0].([]ExecutionHistory)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExecutionHistory indicates an expected call of GetExecutionHistory.
func (mr *MockExecutionStoreMockRecorder) GetExecutionHistory(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExecutionHistory", reflect.TypeOf((*MockExecutionStore)(nil).GetExecutionHistory), ctx, id)
}

// GetExecutions mocks base method.
func (m *MockExecutionStore) GetExecutions(ctx context.Context, jobID string) ([]Execution, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExecutions", ctx, jobID)
	ret0, _ := ret[0].([]Execution)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExecutions indicates an expected call of GetExecutions.
func (mr *MockExecutionStoreMockRecorder) GetExecutions(ctx, jobID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExecutions", reflect.TypeOf((*MockExecutionStore)(nil).GetExecutions), ctx, jobID)
}

// UpdateExecutionState mocks base method.
func (m *MockExecutionStore) UpdateExecutionState(ctx context.Context, request UpdateExecutionStateRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateExecutionState", ctx, request)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateExecutionState indicates an expected call of UpdateExecutionState.
func (mr *MockExecutionStoreMockRecorder) UpdateExecutionState(ctx, request interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateExecutionState", reflect.TypeOf((*MockExecutionStore)(nil).UpdateExecutionState), ctx, request)
}
