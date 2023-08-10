package storage

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// StorageProvider returns a storage that can be used by the job to store data.
type StorageProvider = provider.Provider[Storage]

type Storage interface {
	provider.Providable

	HasStorageLocally(context.Context, models.Artifact) (bool, error)

	// how big is the given volume in terms of resource consumption?
	GetVolumeSize(context.Context, models.Artifact) (uint64, error)

	PrepareStorage(context.Context, models.Artifact) (StorageVolume, error)

	CleanupStorage(context.Context, models.Artifact, StorageVolume) error

	// given a local file path - "store" it and return a SpecConfig
	Upload(context.Context, string) (models.SpecConfig, error)
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
