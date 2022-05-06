package storage

import (
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type PreparedStorageVolume struct {
	Type   string
	Source string
	Target string
}

type StorageProvider interface {
	IsInstalled() (bool, error)
	HasStorage(storageSpec types.StorageSpec) (bool, error)
	PrepareStorage(storageSpec types.StorageSpec) (*PreparedStorageVolume, error)
	CleanupStorage(storageSpec types.StorageSpec, volume *PreparedStorageVolume) error
}
