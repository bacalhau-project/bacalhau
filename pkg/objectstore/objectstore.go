package objectstore

import "context"

// ObjectStore defines the interface to be implemented by data storage
// components
type ObjectStore interface {
	// Get retrieves any data stored in the database with the specified prefix
	// and key.  You can think of the prefix as a bucket, a namespace, or other
	// named container which can group together IDs of a given type (to add an
	// extra level of uniqueness
	Get(ctx context.Context, prefix string, key string) ([]byte, error)

	// GetBatch retrieves one record for each key provided
	GetBatch(ctx context.Context, prefix string, keys []string) (map[string][]byte, error)

	// List returns a list of keys found within the specified prefix
	List(ctx context.Context, prefix string) ([]string, error)

	// Delete removes the value for the specified key from the provided
	// prefix, and if any delete callbacks were registered, will also run
	// those before returning
	Delete(ctx context.Context, prefix string, key string) error

	// Put will store `data` in the prefix namespace/bucket/container with the
	// provided key.  If the Put fails then an error is returned, otherwise
	// nil.
	Put(ctx context.Context, prefix string, key string, value []byte) error

	// Close will close the database, after which it should not be usable
	Close(context.Context) error
}
