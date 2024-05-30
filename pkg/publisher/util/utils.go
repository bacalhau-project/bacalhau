package util

import (
	"context"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	ipfsClient "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/s3"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/tracing"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewPublisherProvider(
	ctx context.Context,
	storagePath string,
	cm *system.CleanupManager,
	cl ipfsClient.Client,
	localConfig *types.LocalPublisherConfig,
) (publisher.PublisherProvider, error) {
	noopPublisher := noop.NewNoopPublisher()
	ipfsPublisher, err := ipfs.NewIPFSPublisher(ctx, cl)
	if err != nil {
		return nil, err
	}

	s3Publisher, err := configureS3Publisher(storagePath, cm)
	if err != nil {
		return nil, err
	}

	localPublisher := local.NewLocalPublisher(ctx, localConfig.Directory, localConfig.Address, localConfig.Port)

	return provider.NewMappedProvider(map[string]publisher.Publisher{
		models.PublisherNoop:  tracing.Wrap(noopPublisher),
		models.PublisherIPFS:  tracing.Wrap(ipfsPublisher),
		models.PublisherS3:    tracing.Wrap(s3Publisher),
		models.PublisherLocal: tracing.Wrap(localPublisher),
	}), nil
}

func configureS3Publisher(storagePath string, cm *system.CleanupManager) (*s3.Publisher, error) {
	dir, err := os.MkdirTemp(storagePath, "bacalhau-s3-publisher")
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
	return provider.NewNoopProvider[publisher.Publisher](noopPublisher), nil
}
