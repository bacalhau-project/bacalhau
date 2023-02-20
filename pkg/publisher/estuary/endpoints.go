package estuary

import (
	"fmt"
	"net/http"

	estuary_client "github.com/application-research/estuary-clients/go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const gatewayEndpoint string = "https://api.estuary.tech"

func getAPIConfig(baseURL string, apiKey string) *estuary_client.Configuration {
	config := estuary_client.NewConfiguration()
	config.BasePath = baseURL
	config.AddDefaultHeader("Authorization", "Bearer "+apiKey)
	config.HTTPClient = &http.Client{
		Transport: otelhttp.NewTransport(nil, otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}), otelhttp.WithSpanOptions(trace.WithAttributes(semconv.PeerService("estuary")))),
	}
	return config
}

func GetClient(apiKey string) *estuary_client.APIClient {
	gatewayConfig := getAPIConfig(gatewayEndpoint, apiKey)
	return estuary_client.NewAPIClient(gatewayConfig)
}
