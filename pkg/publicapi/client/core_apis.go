package client

import (
	"context"
	"io"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

// Alive calls the node's API server health check.
func (apiClient *APIClient) Alive(ctx context.Context) (bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.Alive")
	defer span.End()

	var body io.Reader
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiClient.BaseURI.JoinPath("/api/v1/livez").String(), body)
	if err != nil {
		return false, nil
	}
	res, err := apiClient.Client.Do(req) //nolint:bodyclose // golangcilint is dumb - this is closed
	if err != nil {
		return false, nil
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, "apiClient response", res.Body)

	return res.StatusCode == http.StatusOK, nil
}

func (apiClient *APIClient) Version(ctx context.Context) (*models.BuildVersionInfo, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.Version")
	defer span.End()

	req := legacymodels.VersionRequest{
		ClientID: apiClient.ClientID,
	}

	var res legacymodels.VersionResponse
	if err := apiClient.DoPost(ctx, "/api/v1/version", req, &res); err != nil {
		return nil, err
	}

	return res.VersionInfo, nil
}
