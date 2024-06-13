package localdirectory

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
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

func (driver *StorageProvider) CleanupStorage(_ context.Context, _ models.InputSource, _ storage.StorageVolume) error {
	// We should NOT clean up the storage as it is a locally mounted volume.
	// We are mounting the source directory directly to the target directory and not copying the data.
	return nil
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
