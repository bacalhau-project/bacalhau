package semantic

import (
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

func NewStorageInstalledBidStrategy(storages storage.StorageProvider) bidstrategy.SemanticBidStrategy {
	return NewProviderInstalledArrayStrategy[model.StorageSourceType, storage.Storage](
		storages,
		func(j *model.Job) []model.StorageSourceType {
			var types []model.StorageSourceType
			for _, spec := range j.Spec.AllStorageSpecs() {
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
