package estuary

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/estuary"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"

	"github.com/antihax/optional"
	estuary_client "github.com/application-research/estuary-clients/go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/ipfs/car"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
)

type estuaryPublisher struct {
	client *estuary_client.APIClient
}

const publisherTimeout = 5 * time.Minute

func NewEstuaryPublisher(config EstuaryPublisherConfig) publisher.Publisher {
	return &estuaryPublisher{
		client: GetClient(config.APIKey),
	}
}

// IsInstalled implements publisher.Publisher
func (e *estuaryPublisher) IsInstalled(ctx context.Context) (bool, error) {
	_, response, err := e.client.CollectionsApi.CollectionsGet(ctx) //nolint:bodyclose // golangcilint is dumb - this is closed
	if response != nil {
		defer closer.DrainAndCloseWithLogOnError(ctx, "estuary-response", response.Body)
		return response.StatusCode == http.StatusOK, nil
	} else {
		return false, err
	}
}

func (e *estuaryPublisher) ValidateJob(ctx context.Context, j model.Job) error {
	if j.Spec.PublisherSpec.Type != model.PublisherEstuary {
		return fmt.Errorf("invalid publisher type. expected %s, but received: %s",
			model.PublisherEstuary, j.Spec.PublisherSpec.Type)
	}
	return nil
}

// PublishResult implements publisher.Publisher
func (e *estuaryPublisher) PublishResult(
	ctx context.Context,
	executionID string,
	j model.Job,
	resultPath string,
) (spec.Storage, error) {
	tempDir, err := os.MkdirTemp(os.TempDir(), "bacalhau-estuary-publisher")
	if err != nil {
		return spec.Storage{}, err
	}
	defer os.RemoveAll(tempDir)

	carFile := filepath.Join(tempDir, "results.car")
	_, err = car.CreateCar(ctx, resultPath, carFile, 1)
	if err != nil {
		return spec.Storage{}, err
	}

	carReader, err := os.Open(carFile)
	if err != nil {
		return spec.Storage{}, errors.Wrap(err, "error opening CAR file")
	}

	carContent, err := io.ReadAll(carReader)
	if err != nil {
		return spec.Storage{}, errors.Wrap(err, "error reading CAR data")
	}

	timeout, cancel := context.WithTimeout(ctx, publisherTimeout)
	defer cancel()

	addCarResponse, httpResponse, err := e.client.ContentApi.ContentAddCarPost( //nolint:bodyclose // golangcilint is dumb - this is closed
		timeout,
		string(carContent),
		&estuary_client.ContentApiContentAddCarPostOpts{
			Filename: optional.NewString(j.ID()),
		},
	)
	if err != nil && err != io.EOF {
		return spec.Storage{}, err
	} else if httpResponse.StatusCode != http.StatusOK {
		return spec.Storage{}, fmt.Errorf("upload to Estuary failed")
	}
	log.Ctx(ctx).Debug().Interface("Response", addCarResponse).Int("StatusCode", httpResponse.StatusCode).Msg("Estuary response")
	defer closer.DrainAndCloseWithLogOnError(ctx, "estuary-response", httpResponse.Body)

	respCid, err := cid.Decode(addCarResponse.Cid)
	if err != nil {
		return spec.Storage{}, err
	}
	return (&estuary.EstuaryStorageSpec{
		CID: respCid,
		URL: addCarResponse.EstuaryRetrievalUrl,
	}).AsSpec("TODO", "TODO")
}

func PinToIPFSViaEstuary(
	ctx context.Context,
	EstuaryAPIKey string,
	CID string,
) error {
	client := GetClient(EstuaryAPIKey)
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
