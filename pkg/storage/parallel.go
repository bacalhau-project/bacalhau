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

	out := make([]PreparedStorage, len(specs))
	for i, inputStorageSpec := range specs {
		spec := inputStorageSpec // https://golang.org/doc/faq#closures_and_goroutines
		i := i

		addStorageSpec := func() error {
			var storageProvider Storage
			var volumeMount StorageVolume
			storageProvider, err := provider.Get(ctx, spec.StorageSource)
			if err != nil {
				return err
			}

			volumeMount, err = storageProvider.PrepareStorage(ctx, spec)
			if err != nil {
				return err
			}

			out[i] = PreparedStorage{
				Spec:   spec,
				Volume: volumeMount,
			}
			return nil
		}

		waitgroup.Go(addStorageSpec)
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
