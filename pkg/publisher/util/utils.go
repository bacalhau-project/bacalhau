package util

import (
	"context"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	ipfsClient "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/s3"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/tracing"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

/* TODO(forrest): Fix me(!):
- method has wrong name, this make S3 publisher, and noop, in addition to IPFS
Issue: https://github.com/bacalhau-project/bacalhau/issues/2555

*/

func NewIPFSPublishers(
	ctx context.Context,
	cm *system.CleanupManager,
	cl ipfsClient.Client,
) (publisher.PublisherProvider, error) {
	noopPublisher := noop.NewNoopPublisher()
	ipfsPublisher, err := ipfs.NewIPFSPublisher(ctx, cm, cl)
	if err != nil {
		return nil, err
	}

	s3Publisher, err := configureS3Publisher(cm)
	if err != nil {
		return nil, err
	}
	return model.NewMappedProvider(map[model.Publisher]publisher.Publisher{
		model.PublisherNoop: tracing.Wrap(noopPublisher),
		model.PublisherIpfs: tracing.Wrap(ipfsPublisher),
		model.PublisherS3:   tracing.Wrap(s3Publisher),
	}), nil
}

func configureS3Publisher(cm *system.CleanupManager) (*s3.Publisher, error) {
	dir, err := os.MkdirTemp(config.GetStoragePath(), "bacalhau-s3-publisher")
	if err != nil {
		return nil, err
	}

	cm.RegisterCallback(func() error {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("unable to clean up S3 publisher directory: %w", err)
		}
		return nil
	})

	cfg, err := s3helper.DefaultAWSConfig()
	if err != nil {
		return nil, err
	}
	clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
		AWSConfig: cfg,
	})
	return s3.NewPublisher(s3.PublisherParams{
		LocalDir:       dir,
		ClientProvider: clientProvider,
	}), nil
}

func NewNoopPublishers(
	_ context.Context,
	_ *system.CleanupManager,
	config noop.PublisherConfig,
) (publisher.PublisherProvider, error) {
	noopPublisher := noop.NewNoopPublisherWithConfig(config)
	return model.NewNoopProvider[model.Publisher, publisher.Publisher](noopPublisher), nil
}
