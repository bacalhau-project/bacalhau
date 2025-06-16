package util

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	ipfs_client "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/s3"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/tracing"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewPublisherProvider(
	ctx context.Context,
	cfg types.Bacalhau,
	nclPublisherProvider ncl.PublisherProvider,
) (publisher.PublisherProvider, error) {
	storagePath, err := cfg.ResultsStorageDir()
	if err != nil {
		return nil, err
	}
	providers := make(map[string]publisher.Publisher)

	if cfg.Publishers.IsNotDisabled(models.PublisherNoop) {
		providers[models.PublisherNoop] = noop.NewNoopPublisher()
	}

	if cfg.Publishers.IsNotDisabled(models.PublisherS3) {
		s3Publisher, err := configureS3Publisher(storagePath)
		if err != nil {
			return nil, err
		}
		providers[models.PublisherS3] = tracing.Wrap(s3Publisher)
	}

	if cfg.Publishers.IsNotDisabled(models.PublisherLocal) {
		localPublisher, err := configureLocalPublisher(ctx, cfg, storagePath)
		if err != nil {
			return nil, err
		}
		providers[models.PublisherLocal] = tracing.Wrap(localPublisher)
	}

	if cfg.Publishers.IsNotDisabled(models.PublisherIPFS) {
		if cfg.Publishers.Types.IPFS.Endpoint != "" {
			ipfsClient, err := ipfs_client.NewClient(ctx, cfg.Publishers.Types.IPFS.Endpoint)
			if err != nil {
				return nil, err
			}
			ipfsPublisher, err := ipfs.NewIPFSPublisher(ctx, *ipfsClient)
			if err != nil {
				return nil, err
			}
			providers[models.PublisherIPFS] = tracing.Wrap(ipfsPublisher)
		}
	}

	return provider.NewMappedProvider(providers), nil
}

func configureLocalPublisher(ctx context.Context, cfg types.Bacalhau, storagePath string) (*local.Publisher, error) {
	path := filepath.Join(storagePath, "local-publisher")
	if err := os.MkdirAll(path, util.OS_USER_RWX); err != nil {
		return nil, err
	}
	return local.NewLocalPublisher(
		ctx,
		path,
		cfg.Publishers.Types.Local.Address,
		cfg.Publishers.Types.Local.Port,
	)
}

func configureS3Publisher(storagePath string) (*s3.Publisher, error) {
	path := filepath.Join(storagePath, "s3-publisher")
	if err := os.MkdirAll(path, util.OS_USER_RWX); err != nil {
		return nil, err
	}

	cfg, err := s3helper.DefaultAWSConfig()
	if err != nil {
		return nil, err
	}
	clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
		AWSConfig: cfg,
	})
	return s3.NewPublisher(s3.PublisherParams{
		LocalDir:       path,
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
