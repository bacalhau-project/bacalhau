package estuary

import (
	"context"
	"net/http"

	estuary_client "github.com/application-research/estuary-clients/go"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type EstuaryPinner struct {
	ipfsPublisher publisher.Publisher
	apiClient     *estuary_client.APIClient
}

func NewEstuaryPinner(ctx context.Context, publisher publisher.Publisher, config EstuaryPublisherConfig) (publisher.Publisher, error) {
	client, err := GetGatewayClient(ctx, config.APIKey)
	if err != nil {
		return nil, err
	}

	return &EstuaryPinner{
		ipfsPublisher: publisher,
		apiClient:     client,
	}, nil
}

// IsInstalled implements publisher.Publisher
func (e *EstuaryPinner) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()

	isIpfsInstalled, err := e.ipfsPublisher.IsInstalled(ctx)
	if err != nil || !isIpfsInstalled {
		return isIpfsInstalled, err
	}

	_, response, err := e.apiClient.CollectionsApi.CollectionsGet(ctx)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK, nil
}

// PublishShardResult implements publisher.Publisher
func (e *EstuaryPinner) PublishShardResult(
	ctx context.Context,
	shard model.JobShard,
	hostID string,
	shardResultPath string,
) (model.StorageSpec, error) {
	ctx, span := newSpan(ctx, "PublishShardResult")
	defer span.End()

	// Use IPFS to publish the result.
	log.Ctx(ctx).Debug().Msg("Publishing result to IPFS")
	spec, err := e.ipfsPublisher.PublishShardResult(ctx, shard, hostID, shardResultPath)
	if err != nil {
		return spec, err
	}

	ctx = log.Ctx(ctx).With().
		Str("CID", spec.CID).
		Str("Name", spec.Name).
		Logger().WithContext(ctx)

	// Now pin the CID to Estuary, in a goroutine so this can be slow.
	go func() {
		if spec.CID == "" || spec.Name == "" {
			log.Ctx(ctx).Error().Msgf("Spec %v did not contain a CID or name to pin to Estuary", spec)
		}

		_, response, err := e.apiClient.PinningApi.PinningPinsPost(ctx, estuary_client.TypesIpfsPin{
			Cid:  spec.CID,
			Name: spec.Name,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Bool("Success", false).Msg("Attempted to pin to Estuary")
			return
		}
		defer response.Body.Close()

		success := response.StatusCode == http.StatusAccepted
		level := map[bool]zerolog.Level{true: zerolog.InfoLevel, false: zerolog.ErrorLevel}[success]
		log.Ctx(ctx).WithLevel(level).
			Bool("Success", success).
			Int("ResponseStatusCode", response.StatusCode).
			Msg("Attempted to pin to Estuary")
	}()

	return spec, nil
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "publisher/estuary", apiName)
}

var _ publisher.Publisher = (*EstuaryPinner)(nil)
