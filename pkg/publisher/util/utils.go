package util

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/publisher/estuary"
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
) (map[model.PublisherType]publisher.Publisher, error) {
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
	var estuaryPublisher publisher.Publisher = noopPublisher
	if estuaryAPIKey != "" {
		estuaryPublisher, err = estuary.NewEstuaryPublisher(cm, resolver, estuary.EstuaryPublisherConfig{
			APIKey: estuaryAPIKey,
		})
		if err != nil {
			return nil, err
		}
	}

	return map[model.PublisherType]publisher.Publisher{
		model.PublisherNoop:    noopPublisher,
		model.PublisherIpfs:    ipfsPublisher,
		model.PublisherEstuary: estuaryPublisher,
	}, nil
}

func NewNoopPublishers(
	ctx context.Context,
	cm *system.CleanupManager,
	resolver *job.StateResolver,
) (map[model.PublisherType]publisher.Publisher, error) {
	noopPublisher, err := noop.NewNoopPublisher(ctx, cm, resolver)
	if err != nil {
		return nil, err
	}

	return map[model.PublisherType]publisher.Publisher{
		model.PublisherNoop: noopPublisher,
		model.PublisherIpfs: noopPublisher,
	}, nil
}
