package provider

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"
)

// ConfiguredProvider prevents access to certain values based on a passed in
// block-list of types. It appears as if the disabled types are not installed.
type ConfiguredProvider[Value Providable] struct {
	inner    Provider[Value]
	disabled []string
}

func NewConfiguredProvider[Value Providable](inner Provider[Value], disabled []string) Provider[Value] {
	return &ConfiguredProvider[Value]{inner: inner, disabled: disabled}
}

func (c *ConfiguredProvider[Value]) Get(ctx context.Context, key string) (v Value, err error) {
	if !slices.Contains(c.disabled, key) {
		return c.inner.Get(ctx, key)
	} else {
		return v, fmt.Errorf("%T is disabled: %s", key, key)
	}
}

func (c *ConfiguredProvider[Value]) Has(ctx context.Context, key string) bool {
	if !slices.Contains(c.disabled, key) {
		return c.inner.Has(ctx, key)
	} else {
		return false
	}
}

func (c *ConfiguredProvider[Value]) Keys(ctx context.Context) (keys []string) {
	for _, key := range c.inner.Keys(ctx) {
		if !slices.Contains(c.disabled, key) {
			keys = append(keys, key)
		}
	}
	return
}

// compile-time check that we implement the interface
var _ Provider[Providable] = &ConfiguredProvider[Providable]{}
