package util

import (
	"context"
	"time"

	ipfsClient "github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/publisher/combo"
	"github.com/filecoin-project/bacalhau/pkg/publisher/estuary"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"
	"github.com/filecoin-project/bacalhau/pkg/publisher/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/publisher/noop"
	"github.com/filecoin-project/bacalhau/pkg/publisher/tracing"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func NewIPFSPublishers(
	ctx context.Context,
	cm *system.CleanupManager,
	cl ipfsClient.Client,
	estuaryAPIKey string,
	lotusConfig *filecoinlotus.PublisherConfig,
) (publisher.PublisherProvider, error) {
	defaultPriorityPublisherTimeout := time.Second * 2
	noopPublisher := noop.NewNoopPublisher()
	ipfsPublisher, err := ipfs.NewIPFSPublisher(ctx, cm, cl)
	if err != nil {
		return nil, err
	}

	// we don't want to enforce that every compute node needs to have an estuary API key
	// and so let's only add the
	var estuaryPublisher publisher.Publisher = ipfsPublisher
	if estuaryAPIKey != "" {
		estuaryPublisher = combo.NewPrioritizedFanoutPublisher(
			defaultPriorityPublisherTimeout,
			estuary.NewEstuaryPublisher(estuary.EstuaryPublisherConfig{APIKey: estuaryAPIKey}),
			ipfsPublisher,
		)
		if err != nil {
			return nil, err
		}
	}

	var lotus publisher.Publisher = ipfsPublisher
	if lotusConfig != nil {
		lotus, err = filecoinlotus.NewPublisher(ctx, cm, *lotusConfig)
		if err != nil {
			return nil, err
		}
	}

	return model.NewMappedProvider(map[model.Publisher]publisher.Publisher{
		model.PublisherNoop:     tracing.Wrap(noopPublisher),
		model.PublisherIpfs:     tracing.Wrap(ipfsPublisher),
		model.PublisherEstuary:  tracing.Wrap(estuaryPublisher),
		model.PublisherFilecoin: combo.NewPiggybackedPublisher(tracing.Wrap(ipfsPublisher), tracing.Wrap(lotus)),
	}), nil
}

func NewNoopPublishers(
	_ context.Context,
	_ *system.CleanupManager,
) (publisher.PublisherProvider, error) {
	noopPublisher := noop.NewNoopPublisher()
	return model.NewNoopProvider[model.Publisher, publisher.Publisher](noopPublisher), nil
}
