package localdirectory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type StorageProvider struct {
	LocalDirectoryPath string
}

func NewStorage(_ *system.CleanupManager, localDirectoryPath string) (*StorageProvider, error) {
	storageHandler := &StorageProvider{
		LocalDirectoryPath: localDirectoryPath,
	}
	log.Debug().Msgf("Local directory driver createde: %s", localDirectoryPath)

	// check if the localDirectoryPath exists and error if it doesn't
	if _, err := os.Stat(localDirectoryPath); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("local directory path %s does not exist", localDirectoryPath)
	}

	return storageHandler, nil
}

func (driver *StorageProvider) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (driver *StorageProvider) HasStorageLocally(_ context.Context, volume model.StorageSpec) (bool, error) {
	localPath, err := driver.getPathToVolume(volume)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(localPath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return true, nil
}

func (driver *StorageProvider) GetVolumeSize(_ context.Context, volume model.StorageSpec) (uint64, error) {
	localPath, err := driver.getPathToVolume(volume)
	if err != nil {
		return 0, err
	}
	return util.DirSize(localPath)
}

func (driver *StorageProvider) PrepareStorage(
	_ context.Context,
	storageSpec model.StorageSpec,
) (storage.StorageVolume, error) {
	localPath, err := driver.getPathToVolume(storageSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	return storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: localPath,
		Target: storageSpec.Path,
	}, nil
}

func (driver *StorageProvider) CleanupStorage(context.Context, model.StorageSpec, storage.StorageVolume) error {
	return nil
}

func (driver *StorageProvider) Upload(context.Context, string) (model.StorageSpec, error) {
	return model.StorageSpec{}, fmt.Errorf("not implemented")
}

func (driver *StorageProvider) Explode(_ context.Context, spec model.StorageSpec) ([]model.StorageSpec, error) {
	return []model.StorageSpec{
		spec,
	}, nil
}

func (driver *StorageProvider) getPathToVolume(volume model.StorageSpec) (string, error) {
	// join the driver.LocalDirectoryPath with the volume.SourcePath
	// use the os.PathSeparator to make sure we are using the correct separator for the OS
	localPath := filepath.Join(driver.LocalDirectoryPath, volume.SourcePath)
	return localPath, nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
