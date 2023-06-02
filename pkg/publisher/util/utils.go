package util

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	ipfsClient "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/combo"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/estuary"
	filecoinlotus "github.com/bacalhau-project/bacalhau/pkg/publisher/filecoin_lotus"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/s3"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/tracing"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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

	/*var lotus publisher.Publisher = ipfsPublisher
	if lotusConfig != nil {
		lotus, err = filecoinlotus.NewPublisher(ctx, cm, *lotusConfig)
		if err != nil {
			return nil, err
		}
	}
	*/

	s3Publisher, err := configureS3Publisher(cm)
	if err != nil {
		return nil, err
	}
	return model.NewMappedProvider(map[model.Publisher]publisher.Publisher{
		model.PublisherNoop:    tracing.Wrap(noopPublisher),
		model.PublisherIpfs:    tracing.Wrap(ipfsPublisher),
		model.PublisherS3:      tracing.Wrap(s3Publisher),
		model.PublisherEstuary: tracing.Wrap(estuaryPublisher),
		//model.PublisherFilecoin: combo.NewPiggybackedPublisher(tracing.Wrap(ipfsPublisher), tracing.Wrap(lotus)),
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
