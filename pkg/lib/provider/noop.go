package provider

import (
	"context"
)

// A NoopProvider is a provider that always returns a noop providable regardless of the key
type NoopProvider[Value Providable] struct {
	providable Value
}

// Get implements Provider
func (p *NoopProvider[Value]) Get(ctx context.Context, key string) (v Value, err error) {
	return p.providable, nil
}

// Has implements Provider
func (p *NoopProvider[Value]) Has(ctx context.Context, key string) bool {
	isInstalled, err := p.providable.IsInstalled(ctx)
	return isInstalled && err == nil
}

// Keys implements Provider
func (p *NoopProvider[Value]) Keys(context.Context) []string {
	return []string{"noop"}
}

func NewNoopProvider[Value Providable](providable Value) Provider[Value] {
	return &NoopProvider[Value]{providable: providable}
}

// compile-time check that we implement the interface
var _ Provider[Providable] = &NoopProvider[Providable]{}
