package estuary

import (
	"context"

	estuary_client "github.com/application-research/estuary-clients/go"
)

const gatewayEndpoint string = "https://api.estuary.tech"
const uploadEndpoint string = "https://upload.estuary.tech"

func getAPIConfig(baseURL string, apiKey string) *estuary_client.Configuration {
	config := estuary_client.NewConfiguration()
	config.BasePath = baseURL
	config.AddDefaultHeader("Authorization", "Bearer "+apiKey)
	return config
}

// We need 2 different API endpoints because uploading via the main API URL
// gives a 404 and trying to read via the Upload URL gives a 404 :-(
func GetGatewayClient(ctx context.Context, apiKey string) *estuary_client.APIClient {
	gatewayConfig := getAPIConfig(gatewayEndpoint, apiKey)
	return estuary_client.NewAPIClient(gatewayConfig)
}

func GetUploadClient(ctx context.Context, apiKey string) *estuary_client.APIClient {
	config := getAPIConfig(uploadEndpoint, apiKey)
	return estuary_client.NewAPIClient(config)
}
