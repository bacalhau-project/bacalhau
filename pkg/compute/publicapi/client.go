package publicapi

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// ComputeAPIClient is a utility for interacting with a node's API server.
type ComputeAPIClient struct {
	publicapi.APIClient
}

// NewComputeAPIClient returns a new client for a node's API server.
func NewComputeAPIClient(host string, port uint16, path ...string) *ComputeAPIClient {
	return NewComputeAPIClientFromClient(publicapi.NewAPIClient(host, port, &publicapi.ClientTLSConfig{}, path...))
}

// NewComputeAPIClientFromClient returns a new client for a node's API server.
func NewComputeAPIClientFromClient(baseClient *publicapi.APIClient) *ComputeAPIClient {
	return &ComputeAPIClient{
		APIClient: *baseClient,
	}
}

func (apiClient *ComputeAPIClient) Debug(ctx context.Context) (map[string]model.DebugInfo, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/compute/publicapi.ComputeAPIClient.Debug")
	defer span.End()

	req := struct{}{}
	var res map[string]model.DebugInfo
	if err := apiClient.Post(ctx, APIPrefix+"debug", req, &res); err != nil {
		return res, err
	}

	return res, nil
}
