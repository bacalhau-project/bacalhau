package storage

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/util/generic"
	"go.ptx.dk/multierrgroup"
)

// ParallelPrepareStorage downloads all of the data necessary for the passed
// storage specs in parallel, and returns a map of specs to their download
// volume counterparts.
func ParallelPrepareStorage(
	ctx context.Context,
	provider StorageProvider,
	specs []model.StorageSpec,
) (map[*model.StorageSpec]StorageVolume, error) {
	volumes := generic.SyncMap[*model.StorageSpec, StorageVolume]{}
	waitgroup := multierrgroup.Group{}

	for _, inputStorageSpec := range specs {
		spec := inputStorageSpec // https://golang.org/doc/faq#closures_and_goroutines

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

			volumes.Put(&spec, volumeMount)
			return nil
		}

		waitgroup.Go(addStorageSpec)
	}

	err := waitgroup.Wait()

	returnMap := map[*model.StorageSpec]StorageVolume{}
	volumes.Iter(func(key *model.StorageSpec, value StorageVolume) bool {
		returnMap[key] = value
		return true
	})
	return returnMap, err
}
