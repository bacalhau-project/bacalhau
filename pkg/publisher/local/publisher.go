package local

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"

	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/util/targzip"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type Publisher struct {
	host          string
	urlPrefix     string
	baseDirectory string
	port          int
	server        *LocalPublisherServer
}

func NewLocalPublisher(ctx context.Context, directory string, host string, port int) *Publisher {
	p := &Publisher{
		baseDirectory: directory,
		host:          ResolveAddress(ctx, host),
		port:          port,
		urlPrefix:     fmt.Sprintf("http://%s:%d", host, port),
	}

	p.server = NewLocalPublisherServer(ctx, p.baseDirectory, p.port)
	go p.server.Run(ctx)

	return p
}

// IsInstalled checks if the publisher is installed and it determines
// this based on the presence of the base directory and a local server.
func (p *Publisher) IsInstalled(ctx context.Context) (bool, error) {
	fileInfo, err := os.Stat(p.baseDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			log.Ctx(ctx).Debug().Msg("local publisher not installed because the base directory does not exist")
		} else {
			log.Ctx(ctx).Error().Err(err).Msg("local publisher failed to check if the base directory exists")
		}

		return false, nil
	}

	return fileInfo.IsDir(), nil
}

func (p *Publisher) ValidateJob(ctx context.Context, j models.Job) error {
	return nil
}

func (p *Publisher) PublishResult(
	ctx context.Context, execution *models.Execution, resultPath string) (models.SpecConfig, error) {
	filename := execution.ID + ".tgz"
	targetFile := path.Join(p.baseDirectory, filename)

	file, err := os.Create(targetFile)
	if err != nil {
		return models.SpecConfig{}, errors.Wrap(err, "local publisher failed to create output file")
	}
	defer file.Close()

	writer := io.WriteCloser(file)
	defer closer.CloseWithLogOnError(targetFile, writer)

	if err = targzip.CompressWithoutPath(ctx, resultPath, writer); err != nil {
		return models.SpecConfig{}, errors.Wrap(err, "local publisher failed to compress output file")
	}

	downloadURL, err := url.JoinPath(p.urlPrefix, filename)
	if err != nil {
		return models.SpecConfig{}, errors.Wrap(err, "local publisher failed to generate download URL")
	}

	return models.SpecConfig{
		Type: models.StorageSourceURL,
		Params: map[string]interface{}{
			"URL": downloadURL,
		},
	}, nil
}

var _ publisher.Publisher = (*Publisher)(nil)

func ResolveAddress(ctx context.Context, address string) string {
	addressType, ok := network.AddressTypeFromString(address)
	if ok {
		addrs, err := network.GetNetworkAddress(addressType, network.AllAddresses)
		if err == nil && len(addrs) > 0 {
			return addrs[0]
		} else {
			log.Ctx(ctx).Error().Err(err).Msg("failed to resolve network address by type, using 127.0.0.1")
			return "127.0.0.1"
		}
	}

	return address
}
