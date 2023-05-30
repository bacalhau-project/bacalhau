package localdirectory

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	spec_local "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/local"
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

func (driver *StorageProvider) HasStorageLocally(_ context.Context, volume spec.Storage) (bool, error) {
	localspec, err := spec_local.Decode(volume)
	if err != nil {
		return false, err
	}
	if !driver.isInAllowedPaths(localspec) {
		return false, nil
	}

	if _, err := os.Stat(localspec.Source); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return true, nil
}

func (driver *StorageProvider) GetVolumeSize(_ context.Context, volume spec.Storage) (uint64, error) {
	localspec, err := spec_local.Decode(volume)
	if err != nil {
		return 0, err
	}
	if !driver.isInAllowedPaths(localspec) {
		return 0, errors.New("volume not in allowed paths")
	}
	// check if the volume exists
	if _, err := os.Stat(localspec.Source); errors.Is(err, os.ErrNotExist) {
		return 0, errors.New("volume does not exist")
	}
	// We only query the volume size to make sure we have enough disk space to pull mount the volume locally from a remote location.
	// In this case the data is already local and attempting to query the size would be a waste of time.
	return 0, nil
}

func (driver *StorageProvider) PrepareStorage(
	_ context.Context,
	storageSpec spec.Storage,
) (storage.StorageVolume, error) {
	localspec, err := spec_local.Decode(storageSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	if !driver.isInAllowedPaths(localspec) {
		return storage.StorageVolume{}, errors.New("volume not in allowed paths")
	}
	return storage.StorageVolume{
		Type:     storage.StorageVolumeConnectorBind,
		ReadOnly: !localspec.ReadWrite,
		Source:   localspec.Source,
		Target:   storageSpec.Mount,
	}, nil
}

func (driver *StorageProvider) CleanupStorage(context.Context, spec.Storage, storage.StorageVolume) error {
	return nil
}

func (driver *StorageProvider) Upload(context.Context, string) (spec.Storage, error) {
	return spec.Storage{}, fmt.Errorf("not implemented")
}

func (driver *StorageProvider) isInAllowedPaths(storageSpec *spec_local.LocalStorageSpec) bool {
	for _, allowedPath := range driver.allowedPaths {
		if storageSpec.ReadWrite && !allowedPath.ReadWrite {
			continue
		}
		match, err := doublestar.PathMatch(allowedPath.Path, storageSpec.Source)
		if match && err == nil {
			return true
		}
	}
	return false
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
