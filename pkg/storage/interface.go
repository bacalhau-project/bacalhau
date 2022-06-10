package storage

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/types"
)

type StorageProvider interface {
	IsInstalled(ctx context.Context) (bool, error)

	HasStorage(ctx context.Context, storageSpec types.StorageSpec) (
		bool, error)

	PrepareStorage(ctx context.Context, storageSpec types.StorageSpec) (
		*types.StorageVolume, error)

	CleanupStorage(ctx context.Context, storageSpec types.StorageSpec,
		volume *types.StorageVolume) error
}
