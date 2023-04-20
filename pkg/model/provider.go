package model

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"golang.org/x/exp/slices"
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

// InstalledTypes returns all of the keys which the passed provider has
// installed.
func InstalledTypes[Key ProviderKey, Value Providable](
	ctx context.Context,
	provider Provider[Key, Value],
	allKeys []Key,
) []Key {
	var installedTypes []Key
	for _, key := range allKeys {
		if provider.Has(ctx, key) {
			installedTypes = append(installedTypes, key)
		}
	}
	return installedTypes
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
func (p *NoopProvider[Key, Value]) Has(ctx context.Context, _ Key) bool {
	isInstalled, err := p.noopProvidable.IsInstalled(ctx)
	return isInstalled && err == nil
}

func NewNoopProvider[Key ProviderKey, Value Providable](noopProvidable Value) Provider[Key, Value] {
	return &NoopProvider[Key, Value]{noopProvidable: noopProvidable}
}

// ConfiguredProvider prevents access to certain values based on a passed in
// block-list of types. It appears as if the disabled types are not installed.
type ConfiguredProvider[Key ProviderKey, Value Providable] struct {
	inner    Provider[Key, Value]
	disabled []Key
}

func NewConfiguredProvider[Key ProviderKey, Value Providable](inner Provider[Key, Value], disabled []Key) Provider[Key, Value] {
	return &ConfiguredProvider[Key, Value]{inner: inner, disabled: disabled}
}

func (c *ConfiguredProvider[Key, Value]) Get(ctx context.Context, key Key) (v Value, err error) {
	if !slices.Contains(c.disabled, key) {
		return c.inner.Get(ctx, key)
	} else {
		return v, fmt.Errorf("%T is disabled: %s", key, key)
	}
}

func (c *ConfiguredProvider[Key, Value]) Has(ctx context.Context, key Key) bool {
	if !slices.Contains(c.disabled, key) {
		return c.inner.Has(ctx, key)
	} else {
		return false
	}
}

// ChainedProvider tries multiple different providers when trying to retrieve
// a certain value.
type ChainedProvider[Key ProviderKey, Value Providable] struct {
	Providers []Provider[Key, Value]
}

func (c *ChainedProvider[Key, Value]) Get(ctx context.Context, key Key) (v Value, err error) {
	for idx := range c.Providers {
		v, err = c.Providers[idx].Get(ctx, key)
		if err == nil {
			return
		}
	}
	return v, fmt.Errorf("%T is not installed: %s", key, key)
}

func (c *ChainedProvider[Key, Value]) Has(ctx context.Context, key Key) bool {
	for idx := range c.Providers {
		if c.Providers[idx].Has(ctx, key) {
			return true
		}
	}
	return false
}
