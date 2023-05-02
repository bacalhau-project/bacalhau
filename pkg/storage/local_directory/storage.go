package localdirectory

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/rs/zerolog/log"
)

type StorageProviderParams struct {
	AllowedPaths []string
}
type StorageProvider struct {
	allowedPaths []string
}

func NewStorageProvider(params StorageProviderParams) *StorageProvider {
	storageHandler := &StorageProvider{
		allowedPaths: params.AllowedPaths,
	}
	log.Debug().Msgf("Local directory driver created with allowedPaths: %s", storageHandler.allowedPaths)

	return storageHandler
}

func (driver *StorageProvider) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (driver *StorageProvider) HasStorageLocally(_ context.Context, volume model.StorageSpec) (bool, error) {
	if !driver.isInAllowedPaths(volume.SourcePath) {
		return false, nil
	}

	if _, err := os.Stat(volume.SourcePath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return true, nil
}

func (driver *StorageProvider) GetVolumeSize(_ context.Context, volume model.StorageSpec) (uint64, error) {
	if !driver.isInAllowedPaths(volume.SourcePath) {
		return 0, errors.New("volume not in allowed paths")
	}
	return util.DirSize(volume.SourcePath)
}

func (driver *StorageProvider) PrepareStorage(
	_ context.Context,
	storageSpec model.StorageSpec,
) (storage.StorageVolume, error) {
	if !driver.isInAllowedPaths(storageSpec.SourcePath) {
		return storage.StorageVolume{}, errors.New("volume not in allowed paths")
	}
	return storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: storageSpec.SourcePath,
		Target: storageSpec.Path,
	}, nil
}

func (driver *StorageProvider) CleanupStorage(context.Context, model.StorageSpec, storage.StorageVolume) error {
	return nil
}

func (driver *StorageProvider) Upload(context.Context, string) (model.StorageSpec, error) {
	return model.StorageSpec{}, fmt.Errorf("not implemented")
}

func (driver *StorageProvider) isInAllowedPaths(path string) bool {
	// TODO: check if path is in allowed paths
	return true
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
