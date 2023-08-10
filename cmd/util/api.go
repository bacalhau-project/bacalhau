package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
)

func GetAPIClient(ctx context.Context) *publicapi.RequesterAPIClient {
	return publicapi.NewRequesterAPIClient(config.GetAPIHost(), config.GetAPIPort())
}
