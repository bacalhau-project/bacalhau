package v1beta1

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/util/generic"
)

// A ProviderKey will usually be some lookup value implemented as an enum member
type ProviderKey interface {
	fmt.Stringer
	comparable
}

// A Providable is something that a Provider can check for installation status
type Providable interface {
	IsInstalled(context.Context) (bool, error)
}

// A Provider is an object which is configured to provide certain objects and
// will check their installation status before providing them
type Provider[Key ProviderKey, Value Providable] interface {
	Get(context.Context, Key) (Value, error)
	Has(context.Context, Key) bool
}

// A MappedProvider is a Provider that stores the providables in a simple map,
// and caches permanently the results of checking installation status
type MappedProvider[Key ProviderKey, Value Providable] struct {
	// We can't use a mutex here because it's possible that calling IsInstalled
	// on a Providable might result in calling Get on this Provider (e.g. if the
	// Providable is implemented by composing together other Providables). So we
	// use SyncMaps instead, at the cost of a safe but inefficient race for
	// filling up the installed cache.
	providables    *generic.SyncMap[Key, Value]
	installedCache *generic.SyncMap[Key, bool]
}

func (provider *MappedProvider[Key, Value]) Add(key Key, value Value) {
	provider.providables.Put(key, value)
}

// Get implements Provider
func (provider *MappedProvider[Key, Value]) Get(ctx context.Context, key Key) (v Value, err error) {
	providable, ok := provider.providables.Get(key)
	if !ok {
		return v, fmt.Errorf("no matching %T found on this server: %s", key, key)
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
		return v, fmt.Errorf("%T is not installed: %s", key, key)
	}

	return providable, nil
}

// Has implements Provider
func (provider *MappedProvider[Key, Value]) Has(ctx context.Context, key Key) bool {
	_, err := provider.Get(ctx, key)
	return err == nil
}

func NewMappedProvider[Key ProviderKey, Value Providable](providables map[Key]Value) *MappedProvider[Key, Value] {
	return &MappedProvider[Key, Value]{
		providables:    generic.SyncMapFromMap(providables),
		installedCache: &generic.SyncMap[Key, bool]{},
	}
}

// A NoopProvider is a provider that always returns a singleton providable
// regardless of requested type
type NoopProvider[Key ProviderKey, Value Providable] struct {
	noopProvidable Value
}

// Get implements Provider
func (p *NoopProvider[Key, Value]) Get(context.Context, Key) (Value, error) {
	return p.noopProvidable, nil
}

// Has implements Provider
func (p *NoopProvider[Key, Value]) Has(context.Context, Key) bool {
	return true
}

func NewNoopProvider[Key ProviderKey, Value Providable](noopProvidable Value) Provider[Key, Value] {
	return &NoopProvider[Key, Value]{noopProvidable: noopProvidable}
}
