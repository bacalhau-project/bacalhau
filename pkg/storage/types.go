package storage

import (
	"context"
)

type StorageProvider interface {
	IsInstalled(context.Context) (bool, error)

	HasStorageLocally(context.Context, StorageSpec) (bool, error)

	// how big is the given volume in terms of resource consumption?
	GetVolumeSize(context.Context, StorageSpec) (uint64, error)

	PrepareStorage(context.Context, StorageSpec) (StorageVolume, error)

	CleanupStorage(context.Context, StorageSpec, StorageVolume) error

	// given a local file path - "store" it and return a StorageSpec
	Upload(context.Context, string) (StorageSpec, error)

	// given a StorageSpec - explode it into a list of file paths it contains
	// each file path will be appended to the "path" of the storage spec
	Explode(context.Context, StorageSpec) ([]string, error)
}

// StorageSpec represents some data on a storage engine. Storage engines are
// specific to particular execution engines, as different execution engines
// will mount data in different ways.
type StorageSpec struct {
	// Engine is the execution engine that can mount the spec's data.
	Engine     StorageSourceType `json:"engine,omitempty" yaml:"engine,omitempty"`
	EngineName string            `json:"engine_name" yaml:"engine_name"`

	// Name of the spec's data, for reference.
	Name string `json:"name" yaml:"name"`

	// The unique ID of the data, where it makes sense (for example, in an
	// IPFS storage spec this will be the data's CID).
	Cid string `json:"cid" yaml:"cid"`

	// Source URL of the data
	URL string `json:"url" yaml:"url"`

	// The path that the spec's data should be mounted on, where it makes
	// sense (for example, in a Docker storage spec this will be a filesystem
	// path).
	Path string `json:"path" yaml:"path"`
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
