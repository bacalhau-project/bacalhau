package estuary

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

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
	StateResolver *job.StateResolver
	Config        EstuaryPublisherConfig
}

func NewEstuaryPublisher(
	cm *system.CleanupManager,
	resolver *job.StateResolver,
	config EstuaryPublisherConfig,
) (*EstuaryPublisher, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("APIKey is required")
	}
	return &EstuaryPublisher{
		StateResolver: resolver,
		Config:        config,
	}, nil
}

func (estuaryPublisher *EstuaryPublisher) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()
	_, err := estuaryPublisher.doHTTPRequest("GET", getReadAPIUrl("/content/deals"), nil)
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
	log.Debug().Msgf(
		"Uploading results folder to estuary: %s %s %s",
		hostID,
		shard,
		shardResultPath,
	)
	tempDir, err := ioutil.TempDir("", "bacalhau-estuary-publisher")
	if err != nil {
		return model.StorageSpec{}, err
	}
	carFile := fmt.Sprintf("%s/results.car", tempDir)
	cid, err := car.CreateCar(ctx, shardResultPath, carFile, 1)
	if err != nil {
		return model.StorageSpec{}, err
	}
	fileReader, err := os.Open(carFile)
	if err != nil {
		return model.StorageSpec{}, err
	}
	_, err = estuaryPublisher.doHTTPRequest("POST", getWriteAPIUrl("/content/add-car"), fileReader)
	if err != nil {
		return model.StorageSpec{}, err
	}
	return model.StorageSpec{
		Name:   fmt.Sprintf("job-%s-shard-%d-host-%s", shard.Job.ID, shard.Index, hostID),
		Engine: model.StorageSourceEstuary,
		Cid:    cid,
	}, nil
}

func (estuaryPublisher *EstuaryPublisher) ComposeResultReferences(
	ctx context.Context,
	jobID string,
) ([]model.StorageSpec, error) {
	results := []model.StorageSpec{}
	ctx, span := newSpan(ctx, "ComposeResultSet")
	defer span.End()
	return results, nil
}

func (estuaryPublisher *EstuaryPublisher) doHTTPRequest(
	method string,
	url string,
	body io.Reader,
) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+estuaryPublisher.Config.APIKey)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= http.StatusNotFound {
		return nil, fmt.Errorf("got error response code %d", res.StatusCode)
	}
	return ioutil.ReadAll(res.Body)
}

// We need 2 different API endpoints because uploading via the main API URL
// gives a 404 and trying to read via the Upload URL gives a 404 :-(
func getReadAPIUrl(path string) string {
	baseUrl := os.Getenv("BACALHAU_ESTUARY_READ_API_URL")
	if baseUrl == "" {
		baseUrl = "https://api.estuary.tech"
	}
	return baseUrl + path
}

func getWriteAPIUrl(path string) string {
	baseUrl := os.Getenv("BACALHAU_ESTUARY_WRITE_API_URL")
	if baseUrl == "" {
		baseUrl = "https://shuttle-6.estuary.tech"
	}
	return baseUrl + path
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "publisher/estuary", apiName)
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*EstuaryPublisher)(nil)
