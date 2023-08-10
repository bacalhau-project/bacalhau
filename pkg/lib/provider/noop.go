package provider

import "context"

// A SingletonProvider is a provider that always returns a singleton providable
// regardless of requested type
type SingletonProvider[Value Providable] struct {
	providable Value
}

// Get implements Provider
func (p *SingletonProvider[Value]) Get(context.Context, string) (Value, error) {
	return p.providable, nil
}

// Has implements Provider
func (p *SingletonProvider[Value]) Has(ctx context.Context, _ string) bool {
	isInstalled, err := p.providable.IsInstalled(ctx)
	return isInstalled && err == nil
}

func NewSingletonProvider[Value Providable](providable Value) Provider[Value] {
	return &SingletonProvider[Value]{providable: providable}
}

// compile-time check that we implement the interface
var _ Provider[Providable] = &SingletonProvider[Providable]{}
