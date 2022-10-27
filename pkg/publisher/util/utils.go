package util

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/publisher/combo"
	"github.com/filecoin-project/bacalhau/pkg/publisher/estuary"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"
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
	lotusConfig *filecoinlotus.PublisherConfig,
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
		estuaryPublisher, err = estuary.NewEstuaryPublisher(ctx, cm, resolver, estuary.EstuaryPublisherConfig{
			APIKey: estuaryAPIKey,
		})
		if err != nil {
			return nil, err
		}
	}

	var lotus publisher.Publisher = ipfsPublisher
	if lotusConfig != nil {
		lotus, err = filecoinlotus.NewFilecoinLotusPublisher(ctx, cm, resolver, *lotusConfig)
		if err != nil {
			return nil, err
		}
	}

	return publisher.NewMappedPublisherProvider(map[model.Publisher]publisher.Publisher{
		model.PublisherNoop: noopPublisher,
		model.PublisherIpfs: ipfsPublisher,
		model.PublisherEstuary: combo.NewFallbackPublisher(
			estuaryPublisher,
			ipfsPublisher,
		),
		model.PublisherFilecoin: combo.NewPiggybackedPublisher(ipfsPublisher, lotus),
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
