package util

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
	ipfsClient "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/estuary"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/fanout"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/s3"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/tracing"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

/* TODO(forrest): Fix me(!):
- method has wrong name, this make S3 publisher, estuary, and noop, in addition to IPFS
- is assigning estuary publisher as IPFS publisher and registering it even when not setup
  - I think this will result in the estuary publisher being listed as installed, but really its just IPFS
Issue: https://github.com/bacalhau-project/bacalhau/issues/2555

*/

func NewIPFSPublishers(
	ctx context.Context,
	cm *system.CleanupManager,
	cl ipfsClient.Client,
	estuaryAPIKey string,
) (publisher.PublisherProvider, error) {
	defaultPriorityPublisherTimeout := time.Second * 2
	noopPublisher := noop.NewNoopPublisher()
	ipfsPublisher, err := ipfs.NewIPFSPublisher(ctx, cm, cl)
	if err != nil {
		return nil, err
	}

	// we don't want to enforce that every compute node needs to have an estuary API key
	// and so let's only add the
	// TODO(forrest): this seems like bug, we should not register an estuary publisher if there isn't a key
	// see issue: https://github.com/bacalhau-project/bacalhau/issues/2555
	var estuaryPublisher publisher.Publisher = ipfsPublisher
	if estuaryAPIKey != "" {
		estuaryPublisher = fanout.NewFanoutPublisher(
			[]publisher.Publisher{
				estuary.NewEstuaryPublisher(estuary.EstuaryPublisherConfig{APIKey: estuaryAPIKey}),
				ipfsPublisher,
			},
			fanout.WithPrioritization(),
			fanout.WithTimeout(defaultPriorityPublisherTimeout),
		)
		if err != nil {
			return nil, err
		}
	}

	s3Publisher, err := configureS3Publisher(cm)
	if err != nil {
		return nil, err
	}
	return model.NewMappedProvider(map[model.Publisher]publisher.Publisher{
		model.PublisherNoop:    tracing.Wrap(noopPublisher),
		model.PublisherIpfs:    tracing.Wrap(ipfsPublisher),
		model.PublisherS3:      tracing.Wrap(s3Publisher),
		model.PublisherEstuary: tracing.Wrap(estuaryPublisher),
	}), nil
}

func configureS3Publisher(cm *system.CleanupManager) (*s3.Publisher, error) {
	dir, err := os.MkdirTemp(config_v2.GetStoragePath(), "bacalhau-s3-publisher")
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
