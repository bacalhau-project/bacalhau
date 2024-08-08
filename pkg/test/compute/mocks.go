package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
)

type ManagementEndpointMock struct {
	RegisterHandler        func(ctx context.Context, request requests.RegisterRequest) (*requests.RegisterResponse, error)
	UpdateInfoHandler      func(ctx context.Context, request requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error)
	UpdateResourcesHandler func(ctx context.Context, request requests.UpdateResourcesRequest) (*requests.UpdateResourcesResponse, error)
}

func (m ManagementEndpointMock) Register(ctx context.Context, request requests.RegisterRequest) (*requests.RegisterResponse, error) {
	if m.RegisterHandler != nil {
		return m.RegisterHandler(ctx, request)
	}
	return &requests.RegisterResponse{Accepted: true}, nil
}

func (m ManagementEndpointMock) UpdateInfo(ctx context.Context, request requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error) {
	if m.UpdateInfoHandler != nil {
		return m.UpdateInfoHandler(ctx, request)
	}
	return &requests.UpdateInfoResponse{Accepted: true}, nil
}

func (m ManagementEndpointMock) UpdateResources(
	ctx context.Context, request requests.UpdateResourcesRequest) (*requests.UpdateResourcesResponse, error) {
	if m.UpdateResourcesHandler != nil {
		return m.UpdateResourcesHandler(ctx, request)
	}
	return &requests.UpdateResourcesResponse{}, nil
}

// compile time check if ManagementEndpointMock implements ManagementEndpoint
var _ compute.ManagementEndpoint = ManagementEndpointMock{}

// HeartbeatClientMock is a mock implementation of the HeartbeatClient interface
type HeartbeatClientMock struct {
	SendHeartbeatHandler func(ctx context.Context, sequence uint64) error
}

func (h HeartbeatClientMock) SendHeartbeat(ctx context.Context, sequence uint64) error {
	if h.SendHeartbeatHandler != nil {
		return h.SendHeartbeatHandler(ctx, sequence)
	}
	return nil
}

func (h HeartbeatClientMock) Close(ctx context.Context) error {
	return nil
}

// compile time check if HeartbeatClientMock implements HeartbeatClient
var _ heartbeat.Client = HeartbeatClientMock{}

type CallbackStore struct {
	GetExecutionFn         func(ctx context.Context, id string) (store.LocalExecutionState, error)
	GetExecutionsFn        func(ctx context.Context, id string) ([]store.LocalExecutionState, error)
	GetLiveExecutionsFn    func(ctx context.Context) ([]store.LocalExecutionState, error)
	GetExecutionHistoryFn  func(ctx context.Context, id string) ([]store.LocalStateHistory, error)
	CreateExecutionFn      func(ctx context.Context, execution store.LocalExecutionState) error
	UpdateExecutionStateFn func(ctx context.Context, request store.UpdateExecutionStateRequest) error
	DeleteExecutionFn      func(ctx context.Context, id string) error
	GetExecutionCountFn    func(ctx context.Context, state store.LocalExecutionStateType) (uint64, error)
	GetEventStoreFn        func() watcher.EventStore
	CloseFn                func(ctx context.Context) error
}

func (m *CallbackStore) GetExecution(ctx context.Context, id string) (store.LocalExecutionState, error) {
	return m.GetExecutionFn(ctx, id)
}

func (m *CallbackStore) GetExecutions(ctx context.Context, jobID string) ([]store.LocalExecutionState, error) {
	return m.GetExecutionsFn(ctx, jobID)
}

func (m *CallbackStore) GetLiveExecutions(ctx context.Context) ([]store.LocalExecutionState, error) {
	return m.GetLiveExecutionsFn(ctx)
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

func (m *CallbackStore) GetEventStore() watcher.EventStore {
	return m.GetEventStoreFn()
}
