package provider

import (
	"context"
	"fmt"
)

// A SingletonProvider is a provider that always returns a singleton providable
type SingletonProvider[Value Providable] struct {
	key        string
	providable Value
}

// Get implements Provider
func (p *SingletonProvider[Value]) Get(ctx context.Context, key string) (v Value, err error) {
	if key != p.key {
		err = fmt.Errorf("SingletonProvider only provides %s, but was asked for %s", p.key, key)
		return
	}
	return p.providable, nil
}

// Has implements Provider
func (p *SingletonProvider[Value]) Has(ctx context.Context, key string) bool {
	if key != p.key {
		return false
	}
	isInstalled, err := p.providable.IsInstalled(ctx)
	return isInstalled && err == nil
}

// Keys implements Provider
func (p *SingletonProvider[Value]) Keys(context.Context) []string {
	return []string{p.key}
}

func NewSingletonProvider[Value Providable](key string, providable Value) Provider[Value] {
	return &SingletonProvider[Value]{
		key:        key,
		providable: providable,
	}
}

// compile-time check that we implement the interface
var _ Provider[Providable] = &SingletonProvider[Providable]{}
