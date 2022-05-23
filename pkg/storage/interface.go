package storage

import (
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type StorageProvider interface {
	IsInstalled() (bool, error)
	HasStorage(storageSpec types.StorageSpec) (bool, error)
	PrepareStorage(storageSpec types.StorageSpec) (*types.StorageVolume, error)
	CleanupStorage(storageSpec types.StorageSpec, volume *types.StorageVolume) error
}
