package local

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/util/filecopy"
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
	pathPrefix := path.Join(execution.JobID, execution.ID)
	targetDirectory := filepath.Join(p.baseDirectory, pathPrefix)

	err := filecopy.CopyDir(resultPath, targetDirectory)
	if err != nil {
		return models.SpecConfig{}, errors.Wrap(err, "local publisher failed to publish results")
	}

	downloadURL, err := url.JoinPath(p.urlPrefix, pathPrefix)
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
