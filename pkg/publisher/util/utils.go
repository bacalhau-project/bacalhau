package util

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/publisher/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/publisher/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func NewIPFSPublishers(
	cm *system.CleanupManager,
	ctx context.Context,
	resolver *job.StateResolver,
	ipfsMultiAddress string,
) (map[model.PublisherType]publisher.Publisher, error) {
	noopPublisher, err := noop.NewNoopPublisher(cm, ctx, resolver)
	if err != nil {
		return nil, err
	}

	ipfsPublisher, err := ipfs.NewIPFSPublisher(cm, ctx, resolver, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	return map[model.PublisherType]publisher.Publisher{
		model.PublisherNoop: noopPublisher,
		model.PublisherIpfs: ipfsPublisher,
	}, nil
}

func NewNoopPublishers(
	cm *system.CleanupManager,
	ctx context.Context,
	resolver *job.StateResolver,
) (map[model.PublisherType]publisher.Publisher, error) {
	noopPublisher, err := noop.NewNoopPublisher(cm, ctx, resolver)
	if err != nil {
		return nil, err
	}

	return map[model.PublisherType]publisher.Publisher{
		model.PublisherNoop: noopPublisher,
		model.PublisherIpfs: noopPublisher,
	}, nil
}
