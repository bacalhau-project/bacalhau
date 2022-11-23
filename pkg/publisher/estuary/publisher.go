package estuary

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/antihax/optional"
	estuary_client "github.com/application-research/estuary-clients/go"
	"github.com/filecoin-project/bacalhau/pkg/ipfs/car"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
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
	client := GetGatewayClient(ctx, e.config.APIKey)
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
	_ string,
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

	client := GetUploadClient(ctx, e.config.APIKey)
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

	return model.StorageSpec{
		StorageSource: model.StorageSourceEstuary,
		CID:           addCarResponse.Cid,
	}, nil
}

var _ publisher.Publisher = (*estuaryPublisher)(nil)
