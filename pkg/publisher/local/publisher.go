package local

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

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
		host:          host,
		port:          port,
		urlPrefix:     fmt.Sprintf("http://%s:%d", host, port),
	}

	p.server = NewLocalPublisherServer(ctx, p.baseDirectory, p.host, p.port)
	go p.server.Start(ctx)

	return p
}

// IsInstalled checks if the publisher is installed and it determines
// this based on the presence of the base directory and a local server.
func (p *Publisher) IsInstalled(ctx context.Context) (bool, error) {
	fileInfo, err := os.Stat(p.baseDirectory)
	if err != nil {
		log.Ctx(ctx).Debug().Msg("local publisher not installed because the base directory does not exist")
		return false, nil
	}

	// Check that the HTTP server is running by making a HEAD request to the root route. If the
	// server is not running, or not responding to the HEAD, then we won't be able to serve results
	// and we should not consider the publisher installed.
	resp, err := http.Head(fmt.Sprintf("http://%s:%d/", p.host, p.port)) //nolint:noctx
	if err != nil {
		log.Ctx(ctx).Debug().Msg("local publisher not installed because the server is not running")
		return false, nil
	}
	defer resp.Body.Close() // Required to keep the linter happy even when there is no body.

	if resp.StatusCode != http.StatusOK {
		log.Ctx(ctx).Debug().Msg("local publisher not installed because the server failed to respond")
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
