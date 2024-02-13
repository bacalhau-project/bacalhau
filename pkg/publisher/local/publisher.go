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
}

func NewLocalPublisher(ctx context.Context, directory string, host string, port int) *Publisher {
	return &Publisher{
		baseDirectory: directory,
		host:          host,
		port:          port,
		urlPrefix:     fmt.Sprintf("http://%s:%d", host, port),
	}
}

// IsInstalled checks if the publisher is installed and it determines
// this based on the presence of the base directory.
func (p *Publisher) IsInstalled(ctx context.Context) (bool, error) {
	fileInfo, err := os.Stat(p.baseDirectory)
	if err != nil {
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