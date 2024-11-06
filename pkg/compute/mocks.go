// Code generated by MockGen. DO NOT EDIT.
// Source: types.go

// Package compute is a generated GoMock package.
package compute

import (
	context "context"
	reflect "reflect"

	models "github.com/bacalhau-project/bacalhau/pkg/models"
	messages "github.com/bacalhau-project/bacalhau/pkg/models/messages"
	legacy "github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
	gomock "go.uber.org/mock/gomock"
)

// MockEndpoint is a mock of Endpoint interface.
type MockEndpoint struct {
	ctrl     *gomock.Controller
	recorder *MockEndpointMockRecorder
}

// MockEndpointMockRecorder is the mock recorder for MockEndpoint.
type MockEndpointMockRecorder struct {
	mock *MockEndpoint
}

// NewMockEndpoint creates a new mock instance.
func NewMockEndpoint(ctrl *gomock.Controller) *MockEndpoint {
	mock := &MockEndpoint{ctrl: ctrl}
	mock.recorder = &MockEndpointMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEndpoint) EXPECT() *MockEndpointMockRecorder {
	return m.recorder
}

// AskForBid mocks base method.
func (m *MockEndpoint) AskForBid(arg0 context.Context, arg1 legacy.AskForBidRequest) (legacy.AskForBidResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AskForBid", arg0, arg1)
	ret0, _ := ret[0].(legacy.AskForBidResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AskForBid indicates an expected call of AskForBid.
func (mr *MockEndpointMockRecorder) AskForBid(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AskForBid", reflect.TypeOf((*MockEndpoint)(nil).AskForBid), arg0, arg1)
}

// BidAccepted mocks base method.
func (m *MockEndpoint) BidAccepted(arg0 context.Context, arg1 legacy.BidAcceptedRequest) (legacy.BidAcceptedResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BidAccepted", arg0, arg1)
	ret0, _ := ret[0].(legacy.BidAcceptedResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BidAccepted indicates an expected call of BidAccepted.
func (mr *MockEndpointMockRecorder) BidAccepted(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BidAccepted", reflect.TypeOf((*MockEndpoint)(nil).BidAccepted), arg0, arg1)
}

// BidRejected mocks base method.
func (m *MockEndpoint) BidRejected(arg0 context.Context, arg1 legacy.BidRejectedRequest) (legacy.BidRejectedResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BidRejected", arg0, arg1)
	ret0, _ := ret[0].(legacy.BidRejectedResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BidRejected indicates an expected call of BidRejected.
func (mr *MockEndpointMockRecorder) BidRejected(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BidRejected", reflect.TypeOf((*MockEndpoint)(nil).BidRejected), arg0, arg1)
}

// CancelExecution mocks base method.
func (m *MockEndpoint) CancelExecution(arg0 context.Context, arg1 legacy.CancelExecutionRequest) (legacy.CancelExecutionResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CancelExecution", arg0, arg1)
	ret0, _ := ret[0].(legacy.CancelExecutionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CancelExecution indicates an expected call of CancelExecution.
func (mr *MockEndpointMockRecorder) CancelExecution(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelExecution", reflect.TypeOf((*MockEndpoint)(nil).CancelExecution), arg0, arg1)
}

// MockExecutor is a mock of Executor interface.
type MockExecutor struct {
	ctrl     *gomock.Controller
	recorder *MockExecutorMockRecorder
}

// MockExecutorMockRecorder is the mock recorder for MockExecutor.
type MockExecutorMockRecorder struct {
	mock *MockExecutor
}

// NewMockExecutor creates a new mock instance.
func NewMockExecutor(ctrl *gomock.Controller) *MockExecutor {
	mock := &MockExecutor{ctrl: ctrl}
	mock.recorder = &MockExecutorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExecutor) EXPECT() *MockExecutorMockRecorder {
	return m.recorder
}

// Cancel mocks base method.
func (m *MockExecutor) Cancel(ctx context.Context, execution *models.Execution) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cancel", ctx, execution)
	ret0, _ := ret[0].(error)
	return ret0
}

// Cancel indicates an expected call of Cancel.
func (mr *MockExecutorMockRecorder) Cancel(ctx, execution interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cancel", reflect.TypeOf((*MockExecutor)(nil).Cancel), ctx, execution)
}

// Run mocks base method.
func (m *MockExecutor) Run(ctx context.Context, execution *models.Execution) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", ctx, execution)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockExecutorMockRecorder) Run(ctx, execution interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockExecutor)(nil).Run), ctx, execution)
}

// MockCallback is a mock of Callback interface.
type MockCallback struct {
	ctrl     *gomock.Controller
	recorder *MockCallbackMockRecorder
}

// MockCallbackMockRecorder is the mock recorder for MockCallback.
type MockCallbackMockRecorder struct {
	mock *MockCallback
}

// NewMockCallback creates a new mock instance.
func NewMockCallback(ctrl *gomock.Controller) *MockCallback {
	mock := &MockCallback{ctrl: ctrl}
	mock.recorder = &MockCallbackMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCallback) EXPECT() *MockCallbackMockRecorder {
	return m.recorder
}

// OnBidComplete mocks base method.
func (m *MockCallback) OnBidComplete(ctx context.Context, result legacy.BidResult) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnBidComplete", ctx, result)
}

// OnBidComplete indicates an expected call of OnBidComplete.
func (mr *MockCallbackMockRecorder) OnBidComplete(ctx, result interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnBidComplete", reflect.TypeOf((*MockCallback)(nil).OnBidComplete), ctx, result)
}

// OnComputeFailure mocks base method.
func (m *MockCallback) OnComputeFailure(ctx context.Context, err legacy.ComputeError) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnComputeFailure", ctx, err)
}

// OnComputeFailure indicates an expected call of OnComputeFailure.
func (mr *MockCallbackMockRecorder) OnComputeFailure(ctx, err interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnComputeFailure", reflect.TypeOf((*MockCallback)(nil).OnComputeFailure), ctx, err)
}

// OnRunComplete mocks base method.
func (m *MockCallback) OnRunComplete(ctx context.Context, result legacy.RunResult) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnRunComplete", ctx, result)
}

// OnRunComplete indicates an expected call of OnRunComplete.
func (mr *MockCallbackMockRecorder) OnRunComplete(ctx, result interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnRunComplete", reflect.TypeOf((*MockCallback)(nil).OnRunComplete), ctx, result)
}

// MockManagementEndpoint is a mock of ManagementEndpoint interface.
type MockManagementEndpoint struct {
	ctrl     *gomock.Controller
	recorder *MockManagementEndpointMockRecorder
}

// MockManagementEndpointMockRecorder is the mock recorder for MockManagementEndpoint.
type MockManagementEndpointMockRecorder struct {
	mock *MockManagementEndpoint
}

// NewMockManagementEndpoint creates a new mock instance.
func NewMockManagementEndpoint(ctrl *gomock.Controller) *MockManagementEndpoint {
	mock := &MockManagementEndpoint{ctrl: ctrl}
	mock.recorder = &MockManagementEndpointMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockManagementEndpoint) EXPECT() *MockManagementEndpointMockRecorder {
	return m.recorder
}

// Register mocks base method.
func (m *MockManagementEndpoint) Register(arg0 context.Context, arg1 messages.RegisterRequest) (*messages.RegisterResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Register", arg0, arg1)
	ret0, _ := ret[0].(*messages.RegisterResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Register indicates an expected call of Register.
func (mr *MockManagementEndpointMockRecorder) Register(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Register", reflect.TypeOf((*MockManagementEndpoint)(nil).Register), arg0, arg1)
}

// UpdateInfo mocks base method.
func (m *MockManagementEndpoint) UpdateInfo(arg0 context.Context, arg1 messages.UpdateInfoRequest) (*messages.UpdateInfoResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateInfo", arg0, arg1)
	ret0, _ := ret[0].(*messages.UpdateInfoResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateInfo indicates an expected call of UpdateInfo.
func (mr *MockManagementEndpointMockRecorder) UpdateInfo(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateInfo", reflect.TypeOf((*MockManagementEndpoint)(nil).UpdateInfo), arg0, arg1)
}

// UpdateResources mocks base method.
func (m *MockManagementEndpoint) UpdateResources(arg0 context.Context, arg1 messages.UpdateResourcesRequest) (*messages.UpdateResourcesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateResources", arg0, arg1)
	ret0, _ := ret[0].(*messages.UpdateResourcesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateResources indicates an expected call of UpdateResources.
func (mr *MockManagementEndpointMockRecorder) UpdateResources(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateResources", reflect.TypeOf((*MockManagementEndpoint)(nil).UpdateResources), arg0, arg1)
}
