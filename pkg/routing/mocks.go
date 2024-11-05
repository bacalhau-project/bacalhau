// Code generated by MockGen. DO NOT EDIT.
// Source: types.go

// Package routing is a generated GoMock package.
package routing

import (
	context "context"
	reflect "reflect"

	models "github.com/bacalhau-project/bacalhau/pkg/models"
	gomock "go.uber.org/mock/gomock"
)

// MockNodeInfoStore is a mock of NodeInfoStore interface.
type MockNodeInfoStore struct {
	ctrl     *gomock.Controller
	recorder *MockNodeInfoStoreMockRecorder
}

// MockNodeInfoStoreMockRecorder is the mock recorder for MockNodeInfoStore.
type MockNodeInfoStoreMockRecorder struct {
	mock *MockNodeInfoStore
}

// NewMockNodeInfoStore creates a new mock instance.
func NewMockNodeInfoStore(ctrl *gomock.Controller) *MockNodeInfoStore {
	mock := &MockNodeInfoStore{ctrl: ctrl}
	mock.recorder = &MockNodeInfoStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNodeInfoStore) EXPECT() *MockNodeInfoStoreMockRecorder {
	return m.recorder
}

// Add mocks base method.
func (m *MockNodeInfoStore) Add(ctx context.Context, nodeInfo models.NodeState) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Add", ctx, nodeInfo)
	ret0, _ := ret[0].(error)
	return ret0
}

// Add indicates an expected call of Add.
func (mr *MockNodeInfoStoreMockRecorder) Add(ctx, nodeInfo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*MockNodeInfoStore)(nil).Add), ctx, nodeInfo)
}

// Delete mocks base method.
func (m *MockNodeInfoStore) Delete(ctx context.Context, nodeID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, nodeID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockNodeInfoStoreMockRecorder) Delete(ctx, nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockNodeInfoStore)(nil).Delete), ctx, nodeID)
}

// Get mocks base method.
func (m *MockNodeInfoStore) Get(ctx context.Context, nodeID string) (models.NodeState, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, nodeID)
	ret0, _ := ret[0].(models.NodeState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockNodeInfoStoreMockRecorder) Get(ctx, nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockNodeInfoStore)(nil).Get), ctx, nodeID)
}

// GetByPrefix mocks base method.
func (m *MockNodeInfoStore) GetByPrefix(ctx context.Context, prefix string) (models.NodeState, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByPrefix", ctx, prefix)
	ret0, _ := ret[0].(models.NodeState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByPrefix indicates an expected call of GetByPrefix.
func (mr *MockNodeInfoStoreMockRecorder) GetByPrefix(ctx, prefix interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByPrefix", reflect.TypeOf((*MockNodeInfoStore)(nil).GetByPrefix), ctx, prefix)
}

// List mocks base method.
func (m *MockNodeInfoStore) List(ctx context.Context, filters ...NodeStateFilter) ([]models.NodeState, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range filters {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "List", varargs...)
	ret0, _ := ret[0].([]models.NodeState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockNodeInfoStoreMockRecorder) List(ctx interface{}, filters ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, filters...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockNodeInfoStore)(nil).List), varargs...)
}
