package util

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

func GetAPIClient(ctx context.Context) *client.APIClient {
	return client.NewAPIClient(config.ClientAPIHost(), config.ClientAPIPort())
}

func GetAPIClientV2(ctx context.Context) *clientv2.Client {
	return clientv2.New(clientv2.Options{
		Context: ctx,
		Address: fmt.Sprintf("http://%s:%d", config.ClientAPIHost(), config.ClientAPIPort()),
	})
}
