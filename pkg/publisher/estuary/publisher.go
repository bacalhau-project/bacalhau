package estuary

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"

	"github.com/antihax/optional"
	estuary_client "github.com/application-research/estuary-clients/go"
	"github.com/filecoin-project/bacalhau/pkg/ipfs/car"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type estuaryPublisher struct {
	config EstuaryPublisherConfig
}

const publisherTimeout = 5 * time.Minute

func NewEstuaryPublisher(config EstuaryPublisherConfig) publisher.Publisher {
	return &estuaryPublisher{
		config: config,
	}
}

// IsInstalled implements publisher.Publisher
func (e *estuaryPublisher) IsInstalled(ctx context.Context) (bool, error) {
	client := GetClient(ctx, e.config.APIKey)
	_, response, err := client.CollectionsApi.CollectionsGet(ctx) //nolint:bodyclose // golangcilint is dumb - this is closed
	if response != nil {
		defer closer.DrainAndCloseWithLogOnError(ctx, "estuary-response", response.Body)
		return response.StatusCode == http.StatusOK, nil
	} else {
		return false, err
	}
}

// PublishShardResult implements publisher.Publisher
func (e *estuaryPublisher) PublishShardResult(
	ctx context.Context,
	shard model.JobShard,
	hostID string,
	shardResultPath string,
) (model.StorageSpec, error) {
	tempDir, err := os.MkdirTemp(os.TempDir(), "bacalhau-estuary-publisher")
	if err != nil {
		return model.StorageSpec{}, err
	}
	defer os.RemoveAll(tempDir)

	carFile := filepath.Join(tempDir, "results.car")
	_, err = car.CreateCar(ctx, shardResultPath, carFile, 1)
	if err != nil {
		return model.StorageSpec{}, err
	}

	carReader, err := os.Open(carFile)
	if err != nil {
		return model.StorageSpec{}, errors.Wrap(err, "error opening CAR file")
	}

	carContent, err := io.ReadAll(carReader)
	if err != nil {
		return model.StorageSpec{}, errors.Wrap(err, "error reading CAR data")
	}

	client := GetClient(ctx, e.config.APIKey)
	timeout, cancel := context.WithTimeout(ctx, publisherTimeout)
	defer cancel()

	addCarResponse, httpResponse, err := client.ContentApi.ContentAddCarPost( //nolint:bodyclose // golangcilint is dumb - this is closed
		timeout,
		string(carContent),
		&estuary_client.ContentApiContentAddCarPostOpts{
			Filename: optional.NewString(shard.ID()),
		},
	)
	if err != nil && err != io.EOF {
		return model.StorageSpec{}, err
	} else if httpResponse.StatusCode != http.StatusOK {
		return model.StorageSpec{}, fmt.Errorf("upload to Estuary failed")
	}
	log.Ctx(ctx).Debug().Interface("Response", addCarResponse).Int("StatusCode", httpResponse.StatusCode).Msg("Estuary response")
	defer closer.DrainAndCloseWithLogOnError(ctx, "estuary-response", httpResponse.Body)

	spec := job.GetPublishedStorageSpec(shard, model.StorageSourceEstuary, hostID, addCarResponse.Cid)
	spec.URL = addCarResponse.EstuaryRetrievalUrl

	return spec, nil
}

func PinToIPFSViaEstuary(
	ctx context.Context,
	EstuaryAPIKey string,
	CID string,
) error {
	client := GetClient(ctx, EstuaryAPIKey)
	_, cancel := context.WithTimeout(ctx, publisherTimeout)
	defer cancel()
	pin := estuary_client.TypesIpfsPin{
		Cid: CID,
	}
	addCarResponse, httpResponse, err := client.PinningApi.PinningPinsPost( //nolint:bodyclose // golangcilint is dumb - this is closed
		ctx,
		pin,
	)
	if err != nil && err != io.EOF {
		return err
	} else if httpResponse.StatusCode != http.StatusAccepted {
		return fmt.Errorf("pinning to estuary failed")
	}
	log.Ctx(ctx).Debug().Interface("Response", addCarResponse).Int("StatusCode", httpResponse.StatusCode).Msg("Estuary response")
	defer closer.DrainAndCloseWithLogOnError(ctx, "estuary-response", httpResponse.Body)
	return nil
}

var _ publisher.Publisher = (*estuaryPublisher)(nil)
