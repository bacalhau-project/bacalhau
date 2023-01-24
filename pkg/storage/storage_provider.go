package storage

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/util/generic"
)

// MappedStorageProvider is a simple storage repo that selects a storage based on the job's storage type.
type MappedStorageProvider struct {
	storages               *generic.SyncMap[model.StorageSourceType, Storage]
	storagesInstalledCache *generic.SyncMap[model.StorageSourceType, bool]
}

func NewMappedStorageProvider(storages map[model.StorageSourceType]Storage) *MappedStorageProvider {
	return &MappedStorageProvider{
		storages:               generic.SyncMapFromMap(storages),
		storagesInstalledCache: generic.SyncMapFromMap(map[model.StorageSourceType]bool{}),
	}
}

func (p *MappedStorageProvider) GetStorage(ctx context.Context, storageType model.StorageSourceType) (Storage, error) {
	storage, ok := p.storages.Get(storageType)
	if !ok {
		return nil, fmt.Errorf("no matching storage found on this server: %s", storageType)
	}

	// cache it being installed so we're not hammering it
	// TODO: we should evict the cache in case an installed storage gets uninstalled, or vice versa
	installed, ok := p.storagesInstalledCache.Get(storageType)
	var err error
	if !ok {
		installed, err = storage.IsInstalled(ctx)
		if err != nil {
			return nil, err
		}
		p.storagesInstalledCache.Put(storageType, installed)
	}

	if !installed {
		return nil, fmt.Errorf("storage is not installed: %s", storageType)
	}

	return storage, nil
}

func (p *MappedStorageProvider) HasStorage(ctx context.Context, sourceType model.StorageSourceType) bool {
	_, err := p.GetStorage(ctx, sourceType)
	return err == nil
}
