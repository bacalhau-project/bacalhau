package local

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"

	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/gzip"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
)

type Publisher struct {
	host          string
	urlPrefix     string
	baseDirectory string
	port          int
	server        *LocalPublisherServer
}

func NewLocalPublisher(ctx context.Context, directory string, host string, port int) (*Publisher, error) {
	p := &Publisher{
		baseDirectory: directory,
		// TODO: this field is only written to, never read. It could be deleted.
		host:      ResolveAddress(ctx, host),
		port:      port,
		urlPrefix: fmt.Sprintf("http://%s:%d", host, port),
	}

	var err error
	err = errors.Join(err, validate.NotBlank(p.baseDirectory, "directory cannot be blank"))
	err = errors.Join(err, validate.NotBlank(p.host, "host cannot be blank"))
	err = errors.Join(err, validate.IsGreaterThanZero(p.port, "port must be positive"))
	if err != nil {
		return nil, bacerrors.Wrap(err, "failed to create local publisher").
			WithComponent(errComponent).
			WithCode(bacerrors.ConfigurationError)
	}

	if info, err := os.Stat(p.baseDirectory); err != nil {
		return nil, bacerrors.Wrapf(err, "failed to create local publisher with path %s: path could not be read", directory).
			WithComponent(errComponent).
			WithCode(bacerrors.ConfigurationError)
	} else if !info.IsDir() {
		return nil, bacerrors.Wrapf(err, "failed to create local publisher with path %s: path is not a directory", directory).
			WithComponent(errComponent).
			WithCode(bacerrors.ConfigurationError)
	}
	p.server = NewLocalPublisherServer(ctx, p.baseDirectory, p.port)
	go p.server.Run(ctx)

	return p, nil
}

// IsInstalled checks if the publisher is installed and it determines
// this based on the presence of the base directory and a local server.
func (p *Publisher) IsInstalled(ctx context.Context) (bool, error) {
	fileInfo, err := os.Stat(p.baseDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			log.Ctx(ctx).Error().Err(err).Msgf(
				"local publisher not installed because the base directory %s does not exist", p.baseDirectory)
		} else {
			log.Ctx(ctx).Error().Err(err).Msgf("local publisher failed to check if the base directory %s exists", p.baseDirectory)
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
	filename := execution.ID + ".tar.gz"
	targetFile := path.Join(p.baseDirectory, filename)

	file, err := os.Create(targetFile) //nolint:gosec // G304: targetFile from baseDirectory config, application controlled
	if err != nil {
		return models.SpecConfig{}, pkgerrors.Wrap(err, "local publisher failed to create output file")
	}
	defer func() { _ = file.Close() }()

	err = gzip.Compress(resultPath, file)
	if err != nil {
		return models.SpecConfig{}, pkgerrors.Wrap(err, "local publisher failed to compress output file")
	}

	downloadURL, err := url.JoinPath(p.urlPrefix, filename)
	if err != nil {
		return models.SpecConfig{}, pkgerrors.Wrap(err, "local publisher failed to generate download URL")
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
