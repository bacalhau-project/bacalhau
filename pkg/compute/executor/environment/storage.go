package environment

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

// Mount represents a volume/artifact on disk that has a local
// and hosted component. This is the local path where the
// artifact actually is, and the hosted path where it should
// be mounted by the plugin.
type Mount struct {
	Local  string
	Hosted string

	ReadOnly bool
}

func prepareInputVolumes(ctx context.Context, strgprovider storage.StorageProvider, inputPath string, inputSources ...*models.InputSource) (
	[]storage.PreparedStorage, func(context.Context) error, error) {

	// insert inputPath ...

	inputVolumes, err := storage.ParallelPrepareStorage(ctx, strgprovider, inputSources...)
	if err != nil {
		return nil, nil, err
	}
	return inputVolumes, func(ctx context.Context) error {
		return storage.ParallelCleanStorage(ctx, strgprovider, inputVolumes)
	}, nil
}
