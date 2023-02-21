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

func NewStorage(_ *system.CleanupManager, localPathTemplate string) (*StorageProvider, error) {
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
	var buffer bytes.Buffer
	err := driver.localPathTemplate.Execute(&buffer, volume)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
