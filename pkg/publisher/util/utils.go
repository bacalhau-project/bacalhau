package util

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/publisher/estuary"
	"github.com/filecoin-project/bacalhau/pkg/publisher/fallback"
	"github.com/filecoin-project/bacalhau/pkg/publisher/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/publisher/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func NewIPFSPublishers(
	ctx context.Context,
	cm *system.CleanupManager,
	resolver *job.StateResolver,
	ipfsMultiAddress string,
	estuaryAPIKey string,
) (publisher.PublisherProvider, error) {
	noopPublisher, err := noop.NewNoopPublisher(ctx, cm, resolver)
	if err != nil {
		return nil, err
	}

	ipfsPublisher, err := ipfs.NewIPFSPublisher(ctx, cm, resolver, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	// we don't want to enforce that every compute node needs to have an estuary API key
	// and so let's only add the
	var estuaryPublisher publisher.Publisher = ipfsPublisher
	if estuaryAPIKey != "" {
		estuaryPublisher, err = estuary.NewEstuaryPublisher(cm, resolver, estuary.EstuaryPublisherConfig{
			APIKey: estuaryAPIKey,
		})
		if err != nil {
			return nil, err
		}
	}

	return publisher.NewMappedPublisherProvider(map[model.Publisher]publisher.Publisher{
		model.PublisherNoop: noopPublisher,
		model.PublisherIpfs: ipfsPublisher,
		model.PublisherEstuary: fallback.NewFallbackPublisher(
			estuaryPublisher,
			ipfsPublisher,
		),
	}), nil
}

func NewNoopPublishers(
	ctx context.Context,
	cm *system.CleanupManager,
	resolver *job.StateResolver,
) (publisher.PublisherProvider, error) {
	noopPublisher, err := noop.NewNoopPublisher(ctx, cm, resolver)
	if err != nil {
		return nil, err
	}

	return noop.NewNoopPublisherProvider(noopPublisher), nil
}
