package objectstore

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore/commands"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/distributed"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/local"
)

// ObjectStore defines the interface to be implemented by data storage
// components
type ObjectStore interface {
	// Returns a pointer to CallbackHooks where types can be registered
	CallbackHooks() *commands.CallbackHooks

	// Get retrieves any data stored in the database with the specified prefix
	// and key.  You can think of the prefix as a bucket, a namespace, or other
	// named container which can group together IDs of a given type (to add an
	// extra level of uniqueness
	Get(ctx context.Context, prefix string, key string) ([]byte, error)

	// Put will store `data` in the prefix namespace/bucket/container with the
	// provided key.  If the Put fails then an error is returned, otherwise
	// nil.
	Put(ctx context.Context, prefix string, key string, object any) error

	//
	Close(context.Context)
}

type ImplementationType int

const (
	LocalImplementation ImplementationType = iota
	DistributedImplementation
)

// GetImplementation returns an implementation of the ObjectStore interface
// based on the provided enumeration value. In addition to creating a
// database it will do so using the provided options but it is the callers
// responsibility to ensur they provide the correct options for the given
// ImplementationType.
func GetImplementation(impl ImplementationType, options ...interface{}) (ObjectStore, error) {
	var os ObjectStore

	if impl == LocalImplementation {
		opts, err := convertOptions[local.Option](options...)
		if err != nil {
			return nil, ErrInvalidOption
		}

		os, err = local.New(opts...)
		if err != nil {
			return nil, err
		}
	} else if impl == DistributedImplementation {
		opts, err := convertOptions[distributed.Option](options...)
		if err != nil {
			return nil, ErrInvalidOption
		}

		os, err = distributed.New(opts...)
		if err != nil {
			return nil, err
		}
	}

	return os, nil
}

// Type constraints for the option conversion
type optionImpl interface {
	local.Option | distributed.Option
}

// Converts options of an unknown type to a specific type that we
// can use with the implementations New() function.
func convertOptions[T optionImpl](options ...interface{}) ([]T, error) {
	opts := make([]T, len(options))
	for i, option := range options {
		o, ok := option.(T)
		if !ok {
			return nil, ErrInvalidOption
		}
		opts[i] = o
	}
	return opts, nil
}
