package combo

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"go.opentelemetry.io/otel/trace"
)

type AllProviderFetcher func(ctx context.Context) ([]storage.StorageProvider, error)
type ReadProviderFetcher func(ctx context.Context, spec storage.StorageSpec) (storage.StorageProvider, error)
type WriteProviderFetcher func(ctx context.Context) (storage.StorageProvider, error)

type ComboStorageProvider struct {
	AllFetcher   AllProviderFetcher
	ReadFetcher  ReadProviderFetcher
	WriteFetcher WriteProviderFetcher
}

func NewStorageProvider(
	cm *system.CleanupManager,
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
	_, span := newSpan(ctx, "IsInstalled")
	defer span.End()
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

func (driver *ComboStorageProvider) HasStorageLocally(ctx context.Context, storageSpec storage.StorageSpec) (bool, error) {
	ctx, span := newSpan(ctx, "HasStorageLocally")
	defer span.End()
	provider, err := driver.getReadProvider(ctx, storageSpec)
	if err != nil {
		return false, err
	}
	return provider.HasStorageLocally(ctx, storageSpec)
}

func (driver *ComboStorageProvider) GetVolumeSize(ctx context.Context, storageSpec storage.StorageSpec) (uint64, error) {
	ctx, span := newSpan(ctx, "GetVolumeSize")
	defer span.End()
	provider, err := driver.getReadProvider(ctx, storageSpec)
	if err != nil {
		return 0, err
	}
	return provider.GetVolumeSize(ctx, storageSpec)
}

func (driver *ComboStorageProvider) PrepareStorage(
	ctx context.Context,
	storageSpec storage.StorageSpec,
) (storage.StorageVolume, error) {
	ctx, span := newSpan(ctx, "PrepareStorage")
	defer span.End()
	provider, err := driver.getReadProvider(ctx, storageSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	return provider.PrepareStorage(ctx, storageSpec)
}

func (driver *ComboStorageProvider) CleanupStorage(
	ctx context.Context,
	storageSpec storage.StorageSpec,
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
) (storage.StorageSpec, error) {
	provider, err := driver.getWriteProvider(ctx)
	if err != nil {
		return storage.StorageSpec{}, err
	}
	return provider.Upload(ctx, localPath)
}

func (driver *ComboStorageProvider) Explode(ctx context.Context, storageSpec storage.StorageSpec) ([]storage.StorageSpec, error) {
	provider, err := driver.getReadProvider(ctx, storageSpec)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("no storage provider found for %s", storageSpec.Cid)
	}
	return provider.Explode(ctx, storageSpec)
}

func (driver *ComboStorageProvider) getReadProvider(ctx context.Context, spec storage.StorageSpec) (storage.StorageProvider, error) {
	return driver.ReadFetcher(ctx, spec)
}

func (driver *ComboStorageProvider) getWriteProvider(ctx context.Context) (storage.StorageProvider, error) {
	return driver.WriteFetcher(ctx)
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "storage/combo", apiName)
}

// Compile time interface check:
var _ storage.StorageProvider = (*ComboStorageProvider)(nil)
