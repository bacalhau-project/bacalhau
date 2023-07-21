package storage

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.ptx.dk/multierrgroup"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type PreparedStorage struct {
	Spec   model.StorageSpec
	Volume StorageVolume
}

// ParallelPrepareStorage downloads all of the data necessary for the passed
// storage specs in parallel, and returns a map of specs to their download
// volume counterparts.
func ParallelPrepareStorage(
	ctx context.Context,
	provider StorageProvider,
	specs ...model.StorageSpec,
) ([]PreparedStorage, error) {
	waitgroup := multierrgroup.Group{}

	addStorageSpec := func(idx int, input model.StorageSpec, output []PreparedStorage) error {
		storageProvider, err := provider.Get(ctx, input.StorageSource)
		if err != nil {
			return err
		}

		volumeMount, err := storageProvider.PrepareStorage(ctx, input)
		if err != nil {
			return err
		}

		output[idx] = PreparedStorage{
			Spec:   input,
			Volume: volumeMount,
		}
		return nil
	}

	out := make([]PreparedStorage, len(specs))
	for i, spec := range specs {
		// NB: https://golang.org/doc/faq#closures_and_goroutines
		index := i
		input := spec
		waitgroup.Go(func() error {
			return addStorageSpec(index, input, out)
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
		storage, err := provider.Get(ctx, s.Spec.StorageSource)
		if err != nil {
			log.Ctx(ctx).
				Debug().
				Stringer("Source", s.Spec.StorageSource).
				Msg("failed to get storage provider in cleanup")
			rootErr = errors.Join(rootErr, err)
			continue
		}

		err = storage.CleanupStorage(ctx, s.Spec, s.Volume)
		if err != nil {
			log.Ctx(ctx).
				Debug().
				Stringer("Source", s.Spec.StorageSource).
				Msg("failed to cleanup volume")
			rootErr = errors.Join(rootErr, err)
		}
	}

	return rootErr
}
