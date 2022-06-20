package storage

import (
	"context"
)

type StorageProvider interface {
	IsInstalled(context.Context) (bool, error)

	HasStorage(context.Context, StorageSpec) (bool, error)

	PrepareStorage(context.Context, StorageSpec) (*StorageVolume, error)

	CleanupStorage(context.Context, StorageSpec, *StorageVolume) error
}

// StorageSpec represents some data on a storage engine. Storage engines are
// specific to particular execution engines, as different execution engines
// will mount data in different ways.
type StorageSpec struct {
	// Engine is the execution engine that can mount the spec's data.
	Engine string `json:"engine"`

	// Name of the spec's data, for reference.
	Name string `json:"name"`

	// The unique ID of the data, where it makes sense (for example, in an
	// IPFS storage spec this will be the data's CID).
	Cid string `json:"cid"`

	// The path that the spec's data should be mounted on, where it makes
	// sense (for example, in a Docker storage spec this will be a filesystem
	// path).
	Path string `json:"path"`
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
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}
