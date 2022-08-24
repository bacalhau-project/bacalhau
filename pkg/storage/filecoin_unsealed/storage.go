package filecoin_unsealed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type StorageProvider struct {
	LocalPathTemplateString string
	localPathTemplate       *template.Template
}

func NewStorageProvider(cm *system.CleanupManager, localPathTemplate string) (*StorageProvider, error) {
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
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()
	return true, nil
}

func (driver *StorageProvider) HasStorageLocally(ctx context.Context, volume storage.StorageSpec) (bool, error) {
	ctx, span := newSpan(ctx, "HasStorageLocally")
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

func (driver *StorageProvider) GetVolumeSize(ctx context.Context, volume storage.StorageSpec) (uint64, error) {
	ctx, span := newSpan(ctx, "GetVolumeSize")
	defer span.End()
	localPath, err := driver.getPathToVolume(ctx, volume)
	if err != nil {
		return 0, err
	}
	return dirSize(localPath)
}

func (driver *StorageProvider) PrepareStorage(
	ctx context.Context,
	storageSpec storage.StorageSpec,
) (storage.StorageVolume, error) {
	ctx, span := newSpan(ctx, "PrepareStorage")
	defer span.End()
	var volume storage.StorageVolume
	return volume, nil
}

func (driver *StorageProvider) CleanupStorage(
	ctx context.Context,
	storageSpec storage.StorageSpec,
	volume storage.StorageVolume,
) error {
	return nil
}

func (driver *StorageProvider) Upload(
	ctx context.Context,
	localPath string,
) (storage.StorageSpec, error) {
	return storage.StorageSpec{}, fmt.Errorf("not implemented")
}

func (driver *StorageProvider) Explode(ctx context.Context, spec storage.StorageSpec) ([]storage.StorageSpec, error) {
	// TODO: get a tree of the file system and apply the glob pattern to it
	return []storage.StorageSpec{}, nil
}

func (driver *StorageProvider) getPathToVolume(ctx context.Context, volume storage.StorageSpec) (string, error) {
	var buffer bytes.Buffer
	err := driver.localPathTemplate.Execute(&buffer, volume)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func dirSize(path string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += uint64(info.Size())
		}
		return err
	})
	return size, err
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "storage/ipfs/api_copy", apiName)
}

// Compile time interface check:
var _ storage.StorageProvider = (*StorageProvider)(nil)
