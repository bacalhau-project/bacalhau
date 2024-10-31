package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

type ManagementEndpointMock struct {
	RegisterHandler        func(ctx context.Context, request messages.RegisterRequest) (*messages.RegisterResponse, error)
	UpdateInfoHandler      func(ctx context.Context, request messages.UpdateInfoRequest) (*messages.UpdateInfoResponse, error)
	UpdateResourcesHandler func(ctx context.Context, request messages.UpdateResourcesRequest) (*messages.UpdateResourcesResponse, error)
}

func (m ManagementEndpointMock) Register(ctx context.Context, request messages.RegisterRequest) (*messages.RegisterResponse, error) {
	if m.RegisterHandler != nil {
		return m.RegisterHandler(ctx, request)
	}
	return &messages.RegisterResponse{Accepted: true}, nil
}

func (m ManagementEndpointMock) UpdateInfo(ctx context.Context, request messages.UpdateInfoRequest) (*messages.UpdateInfoResponse, error) {
	if m.UpdateInfoHandler != nil {
		return m.UpdateInfoHandler(ctx, request)
	}
	return &messages.UpdateInfoResponse{Accepted: true}, nil
}

func (m ManagementEndpointMock) UpdateResources(
	ctx context.Context, request messages.UpdateResourcesRequest) (*messages.UpdateResourcesResponse, error) {
	if m.UpdateResourcesHandler != nil {
		return m.UpdateResourcesHandler(ctx, request)
	}
	return &messages.UpdateResourcesResponse{}, nil
}

// compile time check if ManagementEndpointMock implements ManagementEndpoint
var _ compute.ManagementEndpoint = ManagementEndpointMock{}
