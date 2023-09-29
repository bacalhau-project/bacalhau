package storage

import (
	"context"
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/rs/zerolog/log"
	"go.ptx.dk/multierrgroup"
)

type PreparedStorage struct {
	InputSource models.InputSource
	Volume      StorageVolume
}

// ParallelPrepareStorage downloads all of the data necessary for the passed
// storage specs in parallel, and returns a map of specs to their download
// volume counterparts.
func ParallelPrepareStorage(
	ctx context.Context,
	provider StorageProvider,
	specs ...*models.InputSource,
) ([]PreparedStorage, error) {
	waitgroup := multierrgroup.Group{}

	addStorageSpec := func(idx int, input models.InputSource, output []PreparedStorage) error {
		storageProvider, err := provider.Get(ctx, input.Source.Type)
		if err != nil {
			return err
		}

		volumeMount, err := storageProvider.PrepareStorage(ctx, input)
		if err != nil {
			return err
		}

		output[idx] = PreparedStorage{
			InputSource: input,
			Volume:      volumeMount,
		}
		return nil
	}

	out := make([]PreparedStorage, len(specs))
	for i, spec := range specs {
		// NB: https://golang.org/doc/faq#closures_and_goroutines
		index := i
		input := spec
		waitgroup.Go(func() error {
			return addStorageSpec(index, *input, out)
		})
	}
	if err := waitgroup.Wait(); err != nil {
		return nil, err
	}

	return out, nil
}

func ParallelCleanStorage(
	ctx context.Context,
	provider StorageProvider,
	storages []PreparedStorage,
) error {
	var rootErr error

	for _, s := range storages {
		storage, err := provider.Get(ctx, s.InputSource.Source.Type)
		if err != nil {
			log.Ctx(ctx).
				Debug().
				Str("Source", s.InputSource.Source.Type).
				Msg("failed to get storage provider in cleanup")
			rootErr = errors.Join(rootErr, err)
			continue
		}

		err = storage.CleanupStorage(ctx, s.InputSource, s.Volume)
		if err != nil {
			log.Ctx(ctx).
				Debug().
				Str("Source", s.InputSource.Source.Type).
				Msg("failed to cleanup volume")
			rootErr = errors.Join(rootErr, err)
		}
	}

	return rootErr
}
