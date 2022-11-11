package estuary

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/ipfs/car"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type EstuaryPublisherConfig struct {
	APIKey string
}

type EstuaryPublisher struct {
	Config EstuaryPublisherConfig
}

// Partial results from the '/viewer' API endpoint
type EstuaryAPIConfig struct {
	Settings struct {
		ContentAddingDisabled bool
		UploadEndpoints       []string
	}
}

func NewEstuaryPublisher(
	ctx context.Context,
	cm *system.CleanupManager,
	config EstuaryPublisherConfig,
) (*EstuaryPublisher, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("APIKey is required")
	}

	log.Ctx(ctx).Debug().Msgf("Estuary publisher initialized")
	return &EstuaryPublisher{
		Config: config,
	}, nil
}

func (estuaryPublisher *EstuaryPublisher) IsInstalled(ctx context.Context) (bool, error) {
	_, span := newSpan(ctx, "IsInstalled")
	defer span.End()
	_, err := estuaryPublisher.doHTTPRequest(ctx, "GET", getReadAPIURL("/content/deals"), nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (estuaryPublisher *EstuaryPublisher) PublishShardResult(
	ctx context.Context,
	shard model.JobShard,
	hostID string,
	shardResultPath string,
) (model.StorageSpec, error) {
	ctx, span := newSpan(ctx, "PublishShardResult")
	defer span.End()

	log.Ctx(ctx).Info().Msgf("Publishing shard %v results to Estuary", shard)
	tempDir, err := os.MkdirTemp("", "bacalhau-estuary-publisher")
	if err != nil {
		return model.StorageSpec{}, err
	}
	carFile := filepath.Join(tempDir, "results.car")
	cid, err := car.CreateCar(ctx, shardResultPath, carFile, 1)
	if err != nil {
		return model.StorageSpec{}, err
	}

	uploadURLs, err := estuaryPublisher.getWriteAPIURLs(ctx, "/content/add-car")
	if err == nil && len(uploadURLs) < 1 {
		err = fmt.Errorf("cannot upload content because no Estuary servers are available")
	}
	if err != nil {
		return model.StorageSpec{}, err
	}

	// Shuffle the URLs so that we are distributing our work amongst the hosts.
	rand.Shuffle(len(uploadURLs), func(i, j int) {
		uploadURLs[i], uploadURLs[j] = uploadURLs[j], uploadURLs[i]
	})

	// Try each host until one succeeds.
	for _, uploadURL := range uploadURLs {
		fileReader, err := os.Open(carFile)
		if err != nil {
			return model.StorageSpec{}, err
		}
		_, err = estuaryPublisher.doHTTPRequest(ctx, "POST", uploadURL.String(), fileReader)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("Failed to upload to Estuary host '%s'", uploadURL.Host)
			continue
		} else {
			return job.GetPublishedStorageSpec(shard, model.StorageSourceEstuary, hostID, cid), nil
		}
	}

	return model.StorageSpec{}, fmt.Errorf("failed to upload to any Estuary host")
}

func (estuaryPublisher *EstuaryPublisher) doHTTPRequest(
	ctx context.Context,
	method string,
	url string,
	body io.Reader,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+estuaryPublisher.Config.APIKey)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= http.StatusNotFound {
		return nil, fmt.Errorf("got error response code %d", res.StatusCode)
	}
	return io.ReadAll(res.Body)
}

// We need 2 different API endpoints because uploading via the main API URL
// gives a 404 and trying to read via the Upload URL gives a 404 :-(
func getReadAPIURL(path string) string {
	baseURL := os.Getenv("BACALHAU_ESTUARY_READ_API_URL")
	if baseURL == "" {
		baseURL = "https://api.estuary.tech"
	}
	return baseURL + path
}

// getWriteAPIURLs returns a list of URLs that point to different Estuary hosts
// with the given path appended. It uses an Estuary API call to retrieve the
// latest set of write endpoints and checks that Estuary is currently accepting
// writes.
func (estuaryPublisher *EstuaryPublisher) getWriteAPIURLs(ctx context.Context, path string) ([]url.URL, error) {
	baseURL := os.Getenv("BACALHAU_ESTUARY_WRITE_API_URL")
	if baseURL != "" {
		log.Ctx(ctx).Debug().Msgf("Using env-defined '%s' as Estuary upload host", baseURL)
		parsedURL, err := url.Parse(baseURL)
		return []url.URL{*parsedURL}, err
	}

	estuaryConfig, err := estuaryPublisher.doHTTPRequest(ctx, "GET", getReadAPIURL("/viewer"), nil)
	if err != nil {
		return nil, fmt.Errorf("error trying to read Estuary config: %s", err.Error())
	}

	var config EstuaryAPIConfig
	err = model.JSONUnmarshalWithMax(estuaryConfig, &config)
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
			log.Ctx(ctx).Warn().Err(err).Msg("Estuary server URL malformed")
			continue
		}
		parsedURL.Path = path
		uploadURLs = append(uploadURLs, *parsedURL)
	}

	return uploadURLs, nil
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "publisher/estuary", apiName)
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*EstuaryPublisher)(nil)
