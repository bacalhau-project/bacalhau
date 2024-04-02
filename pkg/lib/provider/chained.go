package provider

import (
	"context"
	"fmt"
)

// ChainedProvider tries multiple different providers when trying to retrieve
// a certain value.
type ChainedProvider[Value Providable] struct {
	Providers []Provider[Value]
}

func (c *ChainedProvider[Value]) Get(ctx context.Context, key string) (v Value, err error) {
	for idx := range c.Providers {
		v, err = c.Providers[idx].Get(ctx, key)
		if err == nil {
			return
		}
	}
	return v, fmt.Errorf("%T is not installed: %s", key, key)
}

func (c *ChainedProvider[Value]) Has(ctx context.Context, key string) bool {
	for idx := range c.Providers {
		if c.Providers[idx].Has(ctx, key) {
			return true
		}
	}
	return false
}

func (c *ChainedProvider[Value]) Keys(ctx context.Context) (keys []string) {
	for idx := range c.Providers {
		keys = append(keys, c.Providers[idx].Keys(ctx)...)
	}
	return
}

// compile-time check that we implement the interface
var _ Provider[Providable] = &ChainedProvider[Providable]{}
