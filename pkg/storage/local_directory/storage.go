package localdirectory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type StorageProvider struct {
}

func NewStorage(cm *system.CleanupManager) (*StorageProvider, error) {
	storageHandler := &StorageProvider{}
	log.Debug().Msgf("Local directory driver created")
	return storageHandler, nil
}

func (driver *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (driver *StorageProvider) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/storage/local_directory.HasStorageLocally")
	defer span.End()

	localPath, err := driver.getPathToVolume(ctx, volume)
	if err != nil {
		return false, err
	}

	allowList := os.Getenv("BACALHAU_LOCAL_DIRECTORY_ALLOW_LIST")
	allowPaths := strings.Split(allowList, ",")
	isPathAllowed := false

	for _, allowPath := range allowPaths {
		if strings.HasPrefix(localPath, allowPath) {
			isPathAllowed = true
			break
		}
	}

	if !isPathAllowed {
		return false, fmt.Errorf("path %s is not allowed", localPath)
	}

	if _, err := os.Stat(localPath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return true, nil
}

func (driver *StorageProvider) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/storage/local_directory.GetVolumeSize")
	defer span.End()
	localPath, err := driver.getPathToVolume(ctx, volume)
	if err != nil {
		return 0, err
	}
	return util.DirSize(localPath)
}

func (driver *StorageProvider) PrepareStorage(
	ctx context.Context,
	storageSpec model.StorageSpec,
) (storage.StorageVolume, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/storage/local_directory.PrepareStorage")
	defer span.End()
	localPath, err := driver.getPathToVolume(ctx, storageSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	return storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: localPath,
		Target: storageSpec.Path,
	}, nil
}

func (driver *StorageProvider) CleanupStorage(
	ctx context.Context,
	storageSpec model.StorageSpec,
	volume storage.StorageVolume,
) error {
	return nil
}

func (driver *StorageProvider) Upload(
	ctx context.Context,
	localPath string,
) (model.StorageSpec, error) {
	return model.StorageSpec{}, fmt.Errorf("not implemented")
}

func (driver *StorageProvider) Explode(ctx context.Context, spec model.StorageSpec) ([]model.StorageSpec, error) {
	return []model.StorageSpec{
		spec,
	}, nil
}

func (driver *StorageProvider) getPathToVolume(ctx context.Context, volume model.StorageSpec) (string, error) {
	return volume.SourcePath, nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
