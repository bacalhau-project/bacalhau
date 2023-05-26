package filecoinunsealed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"text/template"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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

func (driver *StorageProvider) HasStorageLocally(_ context.Context, volume spec.Storage) (bool, error) {
	localPath, err := driver.getPathToVolume(volume)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(localPath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return true, nil
}

func (driver *StorageProvider) GetVolumeSize(_ context.Context, volume spec.Storage) (uint64, error) {
	localPath, err := driver.getPathToVolume(volume)
	if err != nil {
		return 0, err
	}
	return util.DirSize(localPath)
}

func (driver *StorageProvider) PrepareStorage(
	_ context.Context,
	storageSpec spec.Storage,
) (storage.StorageVolume, error) {
	localPath, err := driver.getPathToVolume(storageSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	return storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: localPath,
		Target: storageSpec.Mount,
	}, nil
}

func (driver *StorageProvider) CleanupStorage(context.Context, spec.Storage, storage.StorageVolume) error {
	return nil
}

func (driver *StorageProvider) Upload(context.Context, string) (spec.Storage, error) {
	return spec.Storage{}, fmt.Errorf("not implemented")
}

// FIXME(frrist): this will probably break when executing the template
func (driver *StorageProvider) getPathToVolume(volume spec.Storage) (string, error) {
	var buffer bytes.Buffer
	err := driver.localPathTemplate.Execute(&buffer, volume)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
