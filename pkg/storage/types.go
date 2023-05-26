package storage

import (
	"context"

	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
)

// StorageProvider returns a storage that can be used by the job to store data.
type StorageProvider = model.Provider[cid.Cid, Storage]

type Storage interface {
	model.Providable

	HasStorageLocally(context.Context, spec.Storage) (bool, error)

	// how big is the given volume in terms of resource consumption?
	GetVolumeSize(context.Context, spec.Storage) (uint64, error)

	PrepareStorage(context.Context, spec.Storage) (StorageVolume, error)

	CleanupStorage(context.Context, spec.Storage, StorageVolume) error

	// given a local file path - "store" it and return a StorageSpec
	Upload(context.Context, string) (spec.Storage, error)
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
	Type     StorageVolumeConnectorType `json:"type"`
	ReadOnly bool                       `json:"readOnly"`
	Source   string                     `json:"source"`
	Target   string                     `json:"target"`
}
