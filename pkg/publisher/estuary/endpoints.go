package estuary

import (
	"context"

	estuary_client "github.com/application-research/estuary-clients/go"
)

const gatewayEndpoint string = "https://api.estuary.tech"

func getAPIConfig(baseURL string, apiKey string) *estuary_client.Configuration {
	config := estuary_client.NewConfiguration()
	config.BasePath = baseURL
	config.AddDefaultHeader("Authorization", "Bearer "+apiKey)
	return config
}

func GetClient(ctx context.Context, apiKey string) *estuary_client.APIClient {
	gatewayConfig := getAPIConfig(gatewayEndpoint, apiKey)
	return estuary_client.NewAPIClient(gatewayConfig)
}
