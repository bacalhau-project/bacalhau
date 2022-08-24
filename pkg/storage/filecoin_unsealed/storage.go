package filecoin_unsealed

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type StorageProvider struct {
	LocalPathTemplate string
}

func NewStorageProvider(cm *system.CleanupManager, localPathTemplate string) (*StorageProvider, error) {
	storageHandler := &StorageProvider{
		LocalPathTemplate: localPathTemplate,
	}

	log.Debug().Msgf("Filecoin unsealed driver created with path template: %s", localPathTemplate)
	return storageHandler, nil
}

func (driver *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()
	return true, nil
}

func (driver *StorageProvider) HasStorageLocally(ctx context.Context, volume storage.StorageSpec) (bool, error) {
	ctx, span := newSpan(ctx, "HasStorageLocally")
	defer span.End()
	return true, nil
}

func (driver *StorageProvider) GetVolumeSize(ctx context.Context, volume storage.StorageSpec) (uint64, error) {
	ctx, span := newSpan(ctx, "GetVolumeResourceUsage")
	defer span.End()
	return 0, nil
}

func (driver *StorageProvider) PrepareStorage(
	ctx context.Context,
	storageSpec storage.StorageSpec,
) (storage.StorageVolume, error) {
	ctx, span := newSpan(ctx, "PrepareStorage")
	defer span.End()
	var volume storage.StorageVolume
	return volume, nil
}

func (driver *StorageProvider) CleanupStorage(
	ctx context.Context,
	storageSpec storage.StorageSpec,
	volume storage.StorageVolume,
) error {
	return nil
}

func (driver *StorageProvider) Upload(
	ctx context.Context,
	localPath string,
) (storage.StorageSpec, error) {
	return storage.StorageSpec{}, fmt.Errorf("not implemented")
}

func (dockerIPFS *StorageProvider) Explode(ctx context.Context, spec storage.StorageSpec) ([]storage.StorageSpec, error) {
	// TODO: get a tree of the file system and apply the glob pattern to it
	return []storage.StorageSpec{}, nil
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "storage/ipfs/api_copy", apiName)
}

// Compile time interface check:
var _ storage.StorageProvider = (*StorageProvider)(nil)
