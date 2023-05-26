package semantic

import (
	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

func NewStorageInstalledBidStrategy(storages storage.StorageProvider) bidstrategy.SemanticBidStrategy {
	return NewProviderInstalledArrayStrategy(
		storages,
		func(j *model.Job) []cid.Cid {
			// types of storage are represented using the CID of their IPLD schema
			var types []cid.Cid
			for _, spec := range j.Spec.AllStorageSpecs() {
				types = append(types, spec.Schema)
				// If the storage is of invalid type, assume it is unset. It
				// will be caught by the validation process if it is needed.

				// TODO technically we no longer need to validate storage sources since they can be used defined.
				// remove this commented after review
				/*
					if model.IsValidStorageSourceType(spec.StorageSource) {
						types = append(types, spec.StorageSource)
					}
				*/

			}
			return types
		},
	)
}
