package estuary

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"

	estuary_client "github.com/application-research/estuary-clients/go"
	"github.com/rs/zerolog/log"
)

const gatewayEndpoint string = "https://api.estuary.tech"

// Partial results from the '/viewer' API endpoint
type EstuaryAPIConfig struct {
	Settings struct {
		ContentAddingDisabled bool
		UploadEndpoints       []string
	}
}

func getAPIConfig(basePath *url.URL, apiKey string) *estuary_client.Configuration {
	config := estuary_client.NewConfiguration()
	config.BasePath = basePath.String()
	config.AddDefaultHeader("Authorization", "Bearer "+apiKey)
	return config
}

func getGatewayURL() (*url.URL, error) {
	baseURL := os.Getenv("BACALHAU_ESTUARY_READ_API_URL")
	if baseURL == "" {
		baseURL = gatewayEndpoint
	}
	return url.Parse(baseURL)
}

// We need 2 different API endpoints because uploading via the main API URL
// gives a 404 and trying to read via the Upload URL gives a 404 :-(
func GetGatewayClient(ctx context.Context, apiKey string) (*estuary_client.APIClient, error) {
	gatewayURL, err := getGatewayURL()
	if err != nil {
		return nil, err
	}

	gatewayConfig := getAPIConfig(gatewayURL, apiKey)
	return estuary_client.NewAPIClient(gatewayConfig), nil
}

func GetShuttleClients(ctx context.Context, apiKey string) ([]*estuary_client.APIClient, error) {
	gatewayURL, err := getGatewayURL()
	if err != nil {
		return nil, err
	}

	config := getAPIConfig(gatewayURL, apiKey)
	uploadURLs, err := getWriteAPIURLs(ctx, config)
	if err == nil && len(uploadURLs) < 1 {
		err = fmt.Errorf("no Estuary servers are available")
	}
	if err != nil {
		return nil, err
	}

	// Shuffle the URLs so that we are distributing our work amongst the hosts.
	rand.Shuffle(len(uploadURLs), func(i, j int) {
		uploadURLs[i], uploadURLs[j] = uploadURLs[j], uploadURLs[i]
	})

	clients := make([]*estuary_client.APIClient, 0, len(uploadURLs))
	for _, url := range uploadURLs {
		url := url
		config := getAPIConfig(&url, apiKey)
		clients = append(clients, estuary_client.NewAPIClient(config))
	}

	return clients, nil
}

// getWriteAPIURLs returns a list of URLs that point to different Estuary hosts
// with the given path appended. It uses an Estuary API call to retrieve the
// latest set of write endpoints and checks that Estuary is currently accepting
// writes.
func getWriteAPIURLs(ctx context.Context, apiConfig *estuary_client.Configuration) ([]url.URL, error) {
	baseURL := os.Getenv("BACALHAU_ESTUARY_WRITE_API_URL")
	if baseURL != "" {
		log.Ctx(ctx).Debug().Str("Host", baseURL).Msg("Using env-defined Estuary upload host")
		parsedURL, err := url.Parse(baseURL)
		return []url.URL{*parsedURL}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiConfig.BasePath+"/viewier", nil)
	if err != nil {
		return nil, err
	}

	for key, value := range apiConfig.DefaultHeader {
		req.Header.Add(key, value)
	}

	estuaryConfigResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error trying to read Estuary config: %s", err.Error())
	}

	responseBody := make([]byte, estuaryConfigResp.ContentLength)
	bytesRead, err := estuaryConfigResp.Body.Read(responseBody)
	defer estuaryConfigResp.Body.Close()

	if err != nil {
		return nil, err
	}
	if int64(bytesRead) != estuaryConfigResp.ContentLength {
		return nil, fmt.Errorf("read %d bytes but expected %d", bytesRead, estuaryConfigResp.ContentLength)
	}

	var config EstuaryAPIConfig
	err = json.Unmarshal(responseBody, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Estuary config: %s", err.Error())
	}

	if config.Settings.ContentAddingDisabled {
		return nil, fmt.Errorf("cannot upload content because Estuary uploads are disabled")
	}

	uploadURLs := make([]url.URL, len(config.Settings.UploadEndpoints))
	for _, server := range config.Settings.UploadEndpoints {
		parsedURL, err := url.Parse(server)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Str("URL", server).Msg("Estuary server URL malformed")
			continue
		}
		uploadURLs = append(uploadURLs, *parsedURL)
	}

	return uploadURLs, nil
}
