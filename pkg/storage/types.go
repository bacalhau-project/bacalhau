package storage

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// StorageProvider returns a storage that can be used by the job to store data.
type StorageProvider interface {
	model.Provider[model.StorageSourceType, Storage]
}

type Storage interface {
	model.Providable

	HasStorageLocally(context.Context, model.StorageSpec) (bool, error)

	// how big is the given volume in terms of resource consumption?
	GetVolumeSize(context.Context, model.StorageSpec) (uint64, error)

	PrepareStorage(context.Context, model.StorageSpec) (StorageVolume, error)

	CleanupStorage(context.Context, model.StorageSpec, StorageVolume) error

	// given a local file path - "store" it and return a StorageSpec
	Upload(context.Context, string) (model.StorageSpec, error)

	// given a StorageSpec - explode it into a list of storage specs it contains
	// each file path will be appended to the "path" of the storage spec
	Explode(context.Context, model.StorageSpec) ([]model.StorageSpec, error)
}

// a storage entity that is consumed are produced by a job
// input storage specs are turned into storage volumes by drivers
// for example - the input storage spec might be ipfs cid XXX
// and a driver will turn that into a host path that can be consumed by a job
// another example - a wasm storage driver references the upstream ipfs
// cid (source) that can be streamed via a library call using the target name
// put simply - the nature of a storage volume depends on it's use by the
// executor engine
type StorageVolume struct {
	Type   StorageVolumeConnectorType `json:"type"`
	Source string                     `json:"source"`
	Target string                     `json:"target"`
}
