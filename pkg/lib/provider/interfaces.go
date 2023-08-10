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
	Get(context.Context, string) (Value, error)
	Has(context.Context, string) bool
}
