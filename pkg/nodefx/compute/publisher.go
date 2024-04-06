package compute

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/fx"

	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	ipfs_client "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/tracing"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/util"
)

func PublisherProviders(lc fx.Lifecycle, cfg types.PublisherProvidersConfig, client ipfs_client.Client) (publisher.PublisherProvider, error) {
	noopPublisher := noop.NewNoopPublisher()
	ipfsPublisher, err := ipfs.NewIPFSPublisher(client)
	if err != nil {
		return nil, err
	}

	s3Dir, err := os.MkdirTemp(pkgconfig.GetStoragePath(), "bacalhau-s3-publisher")
	s3Publisher, err := util.BetterConfigureS3Publisher(s3Dir)
	if err != nil {
		return nil, err
	}

	localPublisher := local.BetterNewLocalPublisher(cfg.Local.Directory, cfg.Local.Address, cfg.Local.Port)

	pr := provider.NewMappedProvider(map[string]publisher.Publisher{
		// TODO use an fx decorator
		models.PublisherNoop:  tracing.Wrap(noopPublisher),
		models.PublisherIPFS:  tracing.Wrap(ipfsPublisher),
		models.PublisherS3:    tracing.Wrap(s3Publisher),
		models.PublisherLocal: tracing.Wrap(localPublisher),
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// this will stop when the context is cancled
			localPublisher.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := os.RemoveAll(s3Dir); err != nil {
				return fmt.Errorf("unable to clean up S3 publisher directory [%s]: %w", s3Dir, err)
			}
			return nil
		},
	})

	return provider.NewConfiguredProvider(pr, cfg.Disabled), nil

}
