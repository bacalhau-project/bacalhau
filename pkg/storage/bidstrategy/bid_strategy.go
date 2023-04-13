package bidstrategy

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

func NewStorageInstalledBidStrategy(storages storage.StorageProvider) bidstrategy.BidStrategy {
	return bidstrategy.NewProviderInstalledArrayStrategy[model.StorageSourceType, storage.Storage](
		storages,
		func(j *model.Job) []model.StorageSourceType {
			types := []model.StorageSourceType{}
			// TODO(forrest) address this...might get ugly
			storageSpecs, err := j.Spec.AllStorageSpecs()
			if err != nil {
				panic(fmt.Errorf("TODO: %s", err))
			}
			for _, spec := range storageSpecs {
				// If the storage is of invalid type, assume it is unset. It
				// will be caught by the validation process if it is needed.
				if model.IsValidStorageSourceType(spec.StorageSource) {
					types = append(types, spec.StorageSource)
				}
			}
			return types
		},
	)
}
