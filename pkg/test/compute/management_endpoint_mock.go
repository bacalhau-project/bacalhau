package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
)

type ManagementEndpointMock struct {
	RegisterHandler   func(ctx context.Context, request requests.RegisterRequest) (*requests.RegisterResponse, error)
	UpdateInfoHandler func(ctx context.Context, request requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error)
}

func (m ManagementEndpointMock) Register(ctx context.Context, request requests.RegisterRequest) (*requests.RegisterResponse, error) {
	if m.RegisterHandler != nil {
		return m.RegisterHandler(ctx, request)
	}
	return nil, nil
}

func (m ManagementEndpointMock) UpdateInfo(ctx context.Context, request requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error) {
	if m.UpdateInfoHandler != nil {
		return m.UpdateInfoHandler(ctx, request)
	}
	return nil, nil
}
