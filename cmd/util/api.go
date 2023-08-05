package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
)

func GetAPIClient(ctx context.Context) *publicapi.RequesterAPIClient {
	return publicapi.NewRequesterAPIClient(config_v2.GetAPIHost(), config_v2.GetAPIPort())
}
