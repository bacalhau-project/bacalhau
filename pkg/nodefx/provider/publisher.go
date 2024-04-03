package provider

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/s3"
)

func Publisher(cfg map[string][]byte) (publisher.PublisherProvider, error) {
	var (
		provided = make(map[string]publisher.Publisher)
		err      error
	)
	for name, config := range cfg {
		switch strings.ToLower(name) {
		case models.PublisherIPFS:
			provided[name], err = IPFSPublisher(config)
		case models.PublisherS3:
			provided[name], err = S3Publisher(config)
		case models.PublisherLocal:
			provided[name], err = LocalPublisher(config)
		default:
			return nil, fmt.Errorf("unknown publisher provider: %s", name)
		}
		if err != nil {
			return nil, fmt.Errorf("registering %s publisher: %w", name, err)
		}
	}
	return provider.NewMappedProvider(provided), nil
}

func IPFSPublisher(cfg []byte) (*ipfs.IPFSPublisher, error) {
	panic("TODO")
}

func S3Publisher(cfg []byte) (*s3.Publisher, error) {
	panic("TODO")
}

func LocalPublisher(cfg []byte) (*local.Publisher, error) {
	panic("TODO")
}
