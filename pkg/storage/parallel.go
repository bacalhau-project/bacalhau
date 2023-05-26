package storage

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.ptx.dk/multierrgroup"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
)

// ParallelPrepareStorage downloads all of the data necessary for the passed
// storage specs in parallel, and returns a map of specs to their download
// volume counterparts.
func ParallelPrepareStorage(
	ctx context.Context,
	provider StorageProvider,
	specs []spec.Storage,
	// TODO(frrist): dubious usage of points in a map to satisfy comparable constraint. Is the comparable constraint actually
	// needed with the underlying sync.Map used by generic.SyncMap? Or maybe this is all fine.
) (map[*spec.Storage]StorageVolume, error) {
	volumes := generic.SyncMap[*spec.Storage, StorageVolume]{}
	waitgroup := multierrgroup.Group{}

	for _, inputStorageSpec := range specs {
		spec := inputStorageSpec // https://golang.org/doc/faq#closures_and_goroutines

		addStorageSpec := func() error {
			var storageProvider Storage
			var volumeMount StorageVolume
			storageProvider, err := provider.Get(ctx, spec.Schema)
			if err != nil {
				return err
			}

			volumeMount, err = storageProvider.PrepareStorage(ctx, spec)
			if err != nil {
				return err
			}

			volumes.Put(&spec, volumeMount)
			return nil
		}

		waitgroup.Go(addStorageSpec)
	}

	err := waitgroup.Wait()
	if err != nil {
		return nil, err
	}

	// TODO(frrist): dubious usage of points in a map.
	returnMap := map[*spec.Storage]StorageVolume{}
	volumes.Iter(func(key *spec.Storage, value StorageVolume) bool {
		returnMap[key] = value
		return true
	})
	return returnMap, nil
}

func ParallelCleanStorage(
	ctx context.Context,
	provider StorageProvider,
	volumeMap map[*spec.Storage]StorageVolume,
) error {
	var rootErr error

	for storageSpec, storageVolume := range volumeMap {
		storage, err := provider.Get(ctx, storageSpec.Schema)
		if err != nil {
			log.Ctx(ctx).
				Debug().
				Stringer("Source", storageSpec).
				Msg("failed to get storage provider in cleanup")
			rootErr = errors.Join(rootErr, err)
			continue
		}

		err = storage.CleanupStorage(ctx, *storageSpec, storageVolume)
		if err != nil {
			log.Ctx(ctx).
				Debug().
				Stringer("Source", storageSpec).
				Msg("failed to cleanup volume")
			rootErr = errors.Join(rootErr, err)
		}
	}

	return rootErr
}
