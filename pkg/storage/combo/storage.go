package combo

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type AllProviderFetcher func(ctx context.Context) ([]storage.Storage, error)
type ReadProviderFetcher func(ctx context.Context, s spec.Storage) (storage.Storage, error)
type WriteProviderFetcher func(ctx context.Context) (storage.Storage, error)

type ComboStorageProvider struct {
	AllFetcher   AllProviderFetcher
	ReadFetcher  ReadProviderFetcher
	WriteFetcher WriteProviderFetcher
}

func NewStorage(
	_ *system.CleanupManager,
	allFetcher AllProviderFetcher,
	readFetcher ReadProviderFetcher,
	writeFetcher WriteProviderFetcher,
) (*ComboStorageProvider, error) {
	storageHandler := &ComboStorageProvider{
		AllFetcher:   allFetcher,
		ReadFetcher:  readFetcher,
		WriteFetcher: writeFetcher,
	}
	return storageHandler, nil
}

func (driver *ComboStorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	allProviders, err := driver.AllFetcher(ctx)
	if err != nil {
		return false, err
	}
	for _, provider := range allProviders {
		installed, err := provider.IsInstalled(ctx)
		if err != nil {
			return false, err
		}
		if !installed {
			return false, nil
		}
	}
	return true, nil
}

func (driver *ComboStorageProvider) HasStorageLocally(ctx context.Context, storageSpec spec.Storage) (bool, error) {
	provider, err := driver.getReadProvider(ctx, storageSpec)
	if err != nil {
		return false, err
	}
	return provider.HasStorageLocally(ctx, storageSpec)
}

func (driver *ComboStorageProvider) GetVolumeSize(ctx context.Context, storageSpec spec.Storage) (uint64, error) {
	provider, err := driver.getReadProvider(ctx, storageSpec)
	if err != nil {
		return 0, err
	}
	return provider.GetVolumeSize(ctx, storageSpec)
}

func (driver *ComboStorageProvider) PrepareStorage(
	ctx context.Context,
	storageSpec spec.Storage,
) (storage.StorageVolume, error) {
	provider, err := driver.getReadProvider(ctx, storageSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	return provider.PrepareStorage(ctx, storageSpec)
}

func (driver *ComboStorageProvider) CleanupStorage(
	ctx context.Context,
	storageSpec spec.Storage,
	volume storage.StorageVolume,
) error {
	provider, err := driver.getReadProvider(ctx, storageSpec)
	if err != nil {
		return err
	}
	return provider.CleanupStorage(ctx, storageSpec, volume)
}

func (driver *ComboStorageProvider) Upload(
	ctx context.Context,
	localPath string,
) (spec.Storage, error) {
	provider, err := driver.getWriteProvider(ctx)
	if err != nil {
		return spec.Storage{}, err
	}
	return provider.Upload(ctx, localPath)
}

func (driver *ComboStorageProvider) getReadProvider(ctx context.Context, s spec.Storage) (storage.Storage, error) {
	return driver.ReadFetcher(ctx, s)
}

func (driver *ComboStorageProvider) getWriteProvider(ctx context.Context) (storage.Storage, error) {
	return driver.WriteFetcher(ctx)
}

// Compile time interface check:
var _ storage.Storage = (*ComboStorageProvider)(nil)
