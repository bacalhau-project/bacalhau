// Code generated by MockGen. DO NOT EDIT.
// Source: types.go

// Package nclprotocol is a generated GoMock package.
package nclprotocol

import (
	context "context"
	reflect "reflect"

	envelope "github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	watcher "github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	gomock "go.uber.org/mock/gomock"
)

// MockCheckpointer is a mock of Checkpointer interface.
type MockCheckpointer struct {
	ctrl     *gomock.Controller
	recorder *MockCheckpointerMockRecorder
}

// MockCheckpointerMockRecorder is the mock recorder for MockCheckpointer.
type MockCheckpointerMockRecorder struct {
	mock *MockCheckpointer
}

// NewMockCheckpointer creates a new mock instance.
func NewMockCheckpointer(ctrl *gomock.Controller) *MockCheckpointer {
	mock := &MockCheckpointer{ctrl: ctrl}
	mock.recorder = &MockCheckpointerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckpointer) EXPECT() *MockCheckpointerMockRecorder {
	return m.recorder
}

// Checkpoint mocks base method.
func (m *MockCheckpointer) Checkpoint(ctx context.Context, name string, sequenceNumber uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Checkpoint", ctx, name, sequenceNumber)
	ret0, _ := ret[0].(error)
	return ret0
}

// Checkpoint indicates an expected call of Checkpoint.
func (mr *MockCheckpointerMockRecorder) Checkpoint(ctx, name, sequenceNumber interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Checkpoint", reflect.TypeOf((*MockCheckpointer)(nil).Checkpoint), ctx, name, sequenceNumber)
}

// GetCheckpoint mocks base method.
func (m *MockCheckpointer) GetCheckpoint(ctx context.Context, name string) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCheckpoint", ctx, name)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCheckpoint indicates an expected call of GetCheckpoint.
func (mr *MockCheckpointerMockRecorder) GetCheckpoint(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCheckpoint", reflect.TypeOf((*MockCheckpointer)(nil).GetCheckpoint), ctx, name)
}

// MockMessageCreator is a mock of MessageCreator interface.
type MockMessageCreator struct {
	ctrl     *gomock.Controller
	recorder *MockMessageCreatorMockRecorder
}

// MockMessageCreatorMockRecorder is the mock recorder for MockMessageCreator.
type MockMessageCreatorMockRecorder struct {
	mock *MockMessageCreator
}

// NewMockMessageCreator creates a new mock instance.
func NewMockMessageCreator(ctrl *gomock.Controller) *MockMessageCreator {
	mock := &MockMessageCreator{ctrl: ctrl}
	mock.recorder = &MockMessageCreatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMessageCreator) EXPECT() *MockMessageCreatorMockRecorder {
	return m.recorder
}

// CreateMessage mocks base method.
func (m *MockMessageCreator) CreateMessage(event watcher.Event) (*envelope.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateMessage", event)
	ret0, _ := ret[0].(*envelope.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateMessage indicates an expected call of CreateMessage.
func (mr *MockMessageCreatorMockRecorder) CreateMessage(event interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateMessage", reflect.TypeOf((*MockMessageCreator)(nil).CreateMessage), event)
}

// MockMessageCreatorFactory is a mock of MessageCreatorFactory interface.
type MockMessageCreatorFactory struct {
	ctrl     *gomock.Controller
	recorder *MockMessageCreatorFactoryMockRecorder
}

// MockMessageCreatorFactoryMockRecorder is the mock recorder for MockMessageCreatorFactory.
type MockMessageCreatorFactoryMockRecorder struct {
	mock *MockMessageCreatorFactory
}

// NewMockMessageCreatorFactory creates a new mock instance.
func NewMockMessageCreatorFactory(ctrl *gomock.Controller) *MockMessageCreatorFactory {
	mock := &MockMessageCreatorFactory{ctrl: ctrl}
	mock.recorder = &MockMessageCreatorFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMessageCreatorFactory) EXPECT() *MockMessageCreatorFactoryMockRecorder {
	return m.recorder
}

// CreateMessageCreator mocks base method.
func (m *MockMessageCreatorFactory) CreateMessageCreator(ctx context.Context, nodeID string) (MessageCreator, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateMessageCreator", ctx, nodeID)
	ret0, _ := ret[0].(MessageCreator)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateMessageCreator indicates an expected call of CreateMessageCreator.
func (mr *MockMessageCreatorFactoryMockRecorder) CreateMessageCreator(ctx, nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateMessageCreator", reflect.TypeOf((*MockMessageCreatorFactory)(nil).CreateMessageCreator), ctx, nodeID)
}
