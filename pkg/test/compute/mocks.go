package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
)

type ManagementEndpointMock struct {
	RegisterHandler        func(ctx context.Context, request legacy.RegisterRequest) (*legacy.RegisterResponse, error)
	UpdateInfoHandler      func(ctx context.Context, request legacy.UpdateInfoRequest) (*legacy.UpdateInfoResponse, error)
	UpdateResourcesHandler func(ctx context.Context, request legacy.UpdateResourcesRequest) (*legacy.UpdateResourcesResponse, error)
}

func (m ManagementEndpointMock) Register(ctx context.Context, request legacy.RegisterRequest) (*legacy.RegisterResponse, error) {
	if m.RegisterHandler != nil {
		return m.RegisterHandler(ctx, request)
	}
	return &legacy.RegisterResponse{Accepted: true}, nil
}

func (m ManagementEndpointMock) UpdateInfo(ctx context.Context, request legacy.UpdateInfoRequest) (*legacy.UpdateInfoResponse, error) {
	if m.UpdateInfoHandler != nil {
		return m.UpdateInfoHandler(ctx, request)
	}
	return &legacy.UpdateInfoResponse{Accepted: true}, nil
}

func (m ManagementEndpointMock) UpdateResources(
	ctx context.Context, request legacy.UpdateResourcesRequest) (*legacy.UpdateResourcesResponse, error) {
	if m.UpdateResourcesHandler != nil {
		return m.UpdateResourcesHandler(ctx, request)
	}
	return &legacy.UpdateResourcesResponse{}, nil
}

// compile time check if ManagementEndpointMock implements ManagementEndpoint
var _ bprotocol.ManagementEndpoint = ManagementEndpointMock{}
