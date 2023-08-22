package provider

import (
	"context"
)

// A Providable is something that a Provider can check for installation status
type Providable interface {
	IsInstalled(context.Context) (bool, error)
}

// A Provider is an object which is configured to provide certain objects and
// will check their installation status before providing them
type Provider[Value Providable] interface {
	// Get returns the object with the given key
	Get(context.Context, string) (Value, error)
	// Has returns true if the provider can provide the object with the given key
	Has(context.Context, string) bool
	// Keys returns the keys of the objects that the provider can provide
	Keys(context.Context) []string
}
