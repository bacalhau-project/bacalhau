package localdirectory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/rs/zerolog/log"
)

type StorageProviderParams struct {
	AllowedPaths []string
}
type StorageProvider struct {
	allowedPaths []*regexp.Regexp
}

func NewStorageProvider(params StorageProviderParams) (*StorageProvider, error) {
	allowedPaths := make([]*regexp.Regexp, len(params.AllowedPaths))
	for i, path := range params.AllowedPaths {
		allowedPath, err := regexp.Compile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to compile allowed path %s: %w", path, err)
		}
		allowedPaths[i] = allowedPath
	}
	storageHandler := &StorageProvider{
		allowedPaths: allowedPaths,
	}
	log.Debug().Msgf("Local directory driver created with allowedPaths: %s", params.AllowedPaths)

	return storageHandler, nil
}

func (driver *StorageProvider) IsInstalled(context.Context) (bool, error) {
	return len(driver.allowedPaths) > 0, nil
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
	// check if the volume exists
	if _, err := os.Stat(volume.SourcePath); errors.Is(err, os.ErrNotExist) {
		return 0, errors.New("volume does not exist")
	}
	// We only query the volume size to make sure we have enough disk space to pull mount the volume locally from a remote location.
	// In this case the data is already local and attempting to query the size would be a waste of time.
	return 0, nil
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
	for _, allowedPath := range driver.allowedPaths {
		if allowedPath.MatchString(path) {
			return true
		}
	}
	return false
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
