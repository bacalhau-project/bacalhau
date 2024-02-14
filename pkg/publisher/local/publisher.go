package local

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/util/targzip"
	"github.com/pkg/errors"
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
		host:          host,
		port:          port,
		urlPrefix:     fmt.Sprintf("http://%s:%d", host, port),
	}

	p.server = NewLocalPublisherServer(ctx, p.baseDirectory, p.host, p.port)
	go p.server.Start(ctx)

	// var localPublisherServer *local.LocalPublisherServer
	// if pub, err := publishers.Get(ctx, models.PublisherLocal); err == nil {
	// 	ok, err := pub.IsInstalled(ctx)
	// 	if err != nil {
	// 		return nil, errors.Wrap(err, "failed to check if local publisher is installed")
	// 	}
	// 	if ok {
	// 		log.Ctx(ctx).Info().Msg("**** local publisher is installed and server being started")
	// 		localPublisherServer = local.NewLocalPublisherServer(ctx, config.LocalPublisher)
	// 		go localPublisherServer.Start(ctx)
	// 	}
	// } else {
	// 	log.Ctx(ctx).Error().Err(err).Msg("local publisher not installed")
	// }

	return p
}

// IsInstalled checks if the publisher is installed and it determines
// this based on the presence of the base directory and a local server.
func (p *Publisher) IsInstalled(ctx context.Context) (bool, error) {
	fileInfo, err := os.Stat(p.baseDirectory)
	if err != nil {
		return false, nil
	}

	if p.server == nil {
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
