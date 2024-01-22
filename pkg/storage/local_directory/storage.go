package localdirectory

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/rs/zerolog/log"
)

type StorageProviderParams struct {
	AllowedPaths []AllowedPath
}
type StorageProvider struct {
	allowedPaths []AllowedPath
}

func NewStorageProvider(params StorageProviderParams) (*StorageProvider, error) {
	storageHandler := &StorageProvider{
		allowedPaths: params.AllowedPaths,
	}
	log.Debug().Msgf("Local directory driver created with allowedPaths: %s", storageHandler.allowedPaths)

	return storageHandler, nil
}

func (driver *StorageProvider) IsInstalled(context.Context) (bool, error) {
	return len(driver.allowedPaths) > 0, nil
}

func (driver *StorageProvider) HasStorageLocally(_ context.Context, volume models.InputSource) (bool, error) {
	source, err := DecodeSpec(volume.Source)
	if err != nil {
		return false, err
	}
	if !driver.isInAllowedPaths(source) {
		return false, nil
	}

	if _, err := os.Stat(source.SourcePath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return true, nil
}

func (driver *StorageProvider) GetVolumeSize(_ context.Context, volume models.InputSource) (uint64, error) {
	source, err := DecodeSpec(volume.Source)
	if err != nil {
		return 0, err
	}
	if !driver.isInAllowedPaths(source) {
		return 0, errors.New("volume not in allowed paths")
	}
	// check if the volume exists
	if _, err := os.Stat(source.SourcePath); errors.Is(err, os.ErrNotExist) {
		return 0, errors.New("volume does not exist")
	}
	// We only query the volume size to make sure we have enough disk space to pull mount the volume locally from a remote location.
	// In this case the data is already local and attempting to query the size would be a waste of time.
	return 0, nil
}

func (driver *StorageProvider) PrepareStorage(
	_ context.Context,
	_ string,
	storageSpec models.InputSource,
) (storage.StorageVolume, error) {
	source, err := DecodeSpec(storageSpec.Source)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	if !driver.isInAllowedPaths(source) {
		return storage.StorageVolume{}, errors.New("volume not in allowed paths")
	}
	return storage.StorageVolume{
		Type:     storage.StorageVolumeConnectorBind,
		ReadOnly: !source.ReadWrite,
		Source:   source.SourcePath,
		Target:   storageSpec.Target,
	}, nil
}

func (driver *StorageProvider) CleanupStorage(ctx context.Context, src models.InputSource, vol storage.StorageVolume) error {
	return os.Remove(vol.Source)
}

func (driver *StorageProvider) Upload(context.Context, string) (models.SpecConfig, error) {
	return models.SpecConfig{}, fmt.Errorf("not implemented")
}

func (driver *StorageProvider) isInAllowedPaths(storageSpec Source) bool {
	for _, allowedPath := range driver.allowedPaths {
		if storageSpec.ReadWrite && !allowedPath.ReadWrite {
			continue
		}
		match, err := doublestar.PathMatch(allowedPath.Path, storageSpec.SourcePath)
		if match && err == nil {
			return true
		}
	}
	return false
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
