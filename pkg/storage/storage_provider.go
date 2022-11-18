package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// MappedStorageProvider is a simple storage repo that selects a storage based on the job's storage type.
type MappedStorageProvider struct {
	storages               *genericSyncMap[model.StorageSourceType, Storage]
	storagesInstalledCache *genericSyncMap[model.StorageSourceType, bool]
}

func NewMappedStorageProvider(storages map[model.StorageSourceType]Storage) *MappedStorageProvider {
	return &MappedStorageProvider{
		storages:               genericMapFromMap(storages),
		storagesInstalledCache: genericMapFromMap(map[model.StorageSourceType]bool{}),
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

func genericMapFromMap[K comparable, V any](m map[K]V) *genericSyncMap[K, V] {
	ret := &genericSyncMap[K, V]{}
	for k, v := range m {
		ret.Put(k, v)
	}

	return ret
}

type genericSyncMap[K comparable, V any] struct {
	sync.Map
}

func (m *genericSyncMap[K, V]) Get(key K) (V, bool) {
	value, ok := m.Load(key)
	if !ok {
		var empty V
		return empty, false
	}
	return value.(V), true
}

func (m *genericSyncMap[K, V]) Put(key K, value V) {
	m.Store(key, value)
}

func (m *genericSyncMap[K, V]) Iter(ranger func(key K, value V) bool) {
	m.Range(func(key, value any) bool {
		k := key.(K)
		v := value.(V)
		return ranger(k, v)
	})
}
