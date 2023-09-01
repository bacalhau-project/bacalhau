package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
)

func GetAPIClient(ctx context.Context) *client.APIClient {
	return client.NewAPIClient(config.ClientAPIHost(), config.ClientAPIPort())
}
