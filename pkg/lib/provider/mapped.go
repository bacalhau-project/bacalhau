package provider

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
)

// A MappedProvider is a Provider that stores the providables in a simple map,
// and caches permanently the results of checking installation status
type MappedProvider[Value Providable] struct {
	// We can't use a mutex here because it's possible that calling IsInstalled
	// on a Providable might result in calling Get on this Provider (e.g. if the
	// Providable is implemented by composing together other Providables). So we
	// use SyncMaps instead, at the cost of a safe but inefficient race for
	// filling up the installed cache.
	providables    *generic.SyncMap[string, Value]
	installedCache *generic.SyncMap[string, bool]
}

func (provider *MappedProvider[Value]) Add(key string, value Value) {
	provider.providables.Put(sanitizeKey(key), value)
}

// Get implements Provider
func (provider *MappedProvider[Value]) Get(ctx context.Context, key string) (v Value, err error) {
	key = sanitizeKey(key)
	providable, ok := provider.providables.Get(key)
	if !ok {
		return v, fmt.Errorf("no matching key found on this server: %s. Only supports %s", key, provider.providables.Keys())
	}

	// cache it being installed so we're not hammering it. TODO: we should evict
	// the cache in case an installed providable gets uninstalled, or vice versa
	installed, ok := provider.installedCache.Get(key)
	if !ok {
		installed, err = providable.IsInstalled(ctx)
		if err != nil {
			return v, err
		}
		provider.installedCache.Put(key, installed)
	}

	if !installed {
		return v, fmt.Errorf("key is not installed: %s", key)
	}

	return providable, nil
}

// Has implements Provider
func (provider *MappedProvider[Value]) Has(ctx context.Context, key string) bool {
	_, err := provider.Get(ctx, sanitizeKey(key))
	return err == nil
}

func (provider *MappedProvider[Value]) Keys(ctx context.Context) (keys []string) {
	provider.providables.Range(func(k, _ any) bool {
		key := k.(string)
		if provider.Has(ctx, key) {
			keys = append(keys, key)
		}
		return true
	})
	return
}

func NewMappedProvider[Value Providable](providables map[string]Value) *MappedProvider[Value] {
	p := &MappedProvider[Value]{
		providables:    &generic.SyncMap[string, Value]{},
		installedCache: &generic.SyncMap[string, bool]{},
	}
	for k, v := range providables {
		p.Add(k, v)
	}
	return p
}

// compile-time check that we implement the interface
var _ Provider[Providable] = &MappedProvider[Providable]{}
