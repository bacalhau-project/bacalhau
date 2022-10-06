package storage

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// A simple storage repo that selects a storage based on the job's storage type.
type MappedStorageProvider struct {
	storages               map[model.StorageSourceType]Storage
	storagesInstalledCache map[model.StorageSourceType]bool
}

func NewMappedStorageProvider(storages map[model.StorageSourceType]Storage) *MappedStorageProvider {
	return &MappedStorageProvider{
		storages:               storages,
		storagesInstalledCache: map[model.StorageSourceType]bool{},
	}
}

func (p *MappedStorageProvider) GetStorage(ctx context.Context, storageType model.StorageSourceType) (Storage, error) {
	storage, ok := p.storages[storageType]
	if !ok {
		return nil, fmt.Errorf(
			"no matching storage found on this server: %s", storageType)
	}

	// cache it being installed so we're not hammering it
	// TODO: we should evict the cache in case an installed storage gets uninstalled, or vice versa
	installed, ok := p.storagesInstalledCache[storageType]
	var err error
	if !ok {
		installed, err = storage.IsInstalled(ctx)
		if err != nil {
			return nil, err
		}
		p.storagesInstalledCache[storageType] = installed
	}

	if !installed {
		return nil, fmt.Errorf("storage is not installed: %s", storageType)
	}

	return storage, nil
}
