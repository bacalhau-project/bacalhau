package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
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
