package filecoinunsealed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"text/template"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type StorageProvider struct {
	LocalPathTemplateString string
	localPathTemplate       *template.Template
}

func NewStorage(cm *system.CleanupManager, localPathTemplate string) (*StorageProvider, error) {
	t := template.New("bacalhau-storage-filecoin-unsealed-path")
	t, err := t.Parse(localPathTemplate)
	if err != nil {
		return nil, err
	}
	storageHandler := &StorageProvider{
		LocalPathTemplateString: localPathTemplate,
		localPathTemplate:       t,
	}
	log.Debug().Msgf("Filecoin unsealed driver created with path template: %s", localPathTemplate)
	return storageHandler, nil
}

func (driver *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (driver *StorageProvider) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/storage/filecoin_unsealed.HasStorageLocally")
	defer span.End()

	localPath, err := driver.getPathToVolume(ctx, volume)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(localPath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return true, nil
}

func (driver *StorageProvider) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/storage/filecoin_unsealed.GetVolumeSize")
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
	ctx, span := system.GetTracer().Start(ctx, "pkg/storage/filecoin_unsealed.PrepareStorage")
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
	var buffer bytes.Buffer
	err := driver.localPathTemplate.Execute(&buffer, volume)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
