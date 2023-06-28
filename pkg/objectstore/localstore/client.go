package localstore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/rs/zerolog/log"
)

type indexable = objectstore.Indexable

type Client[T indexable] struct {
	ctx    context.Context
	newT   func() T // Used to create a new empty T
	prefix string
	store  *LocalStore
}

// NewClient creates a new, typed, client for interacting with a LocalStore instance.
// It is constrained to a single type, within a single prefix, but is still able to use
// other prefixes where they originate with an indexer.
func NewClient[T indexable](ctx context.Context, prefix string, store *LocalStore) *Client[T] {
	log.Ctx(ctx).Debug().
		Str("Type", fmt.Sprintf("%T", *new(T))).
		Msg("creating new localstore client")

	return &Client[T]{
		ctx:    ctx,
		newT:   func() T { return *new(T) },
		prefix: prefix,
		store:  store,
	}
}

// Get will return a T for the provided key, or will return an empty T and
// an ErrNotFound error.
func (c *Client[T]) Get(key string) (T, error) {
	t := c.newT()

	log.Ctx(c.ctx).Debug().
		Str("Key", key).
		Msg("localstore.client.Get")

	bytes, err := c.store.Get(c.ctx, c.prefix, key)
	if err != nil {
		return t, objectstore.NewErrNotFound(key)
	}

	err = json.Unmarshal(bytes, &t)
	if err != nil {
		return t, nil
	}

	return t, nil
}

// Put will write the provided T under the provided K, or will return an
// error if it is unable to do so. Once the item has been written this
// method will ask the object for its indexers which will be executed
// in sequence.
func (c *Client[T]) Put(key string, object T) error {
	log.Ctx(c.ctx).Debug().
		Str("Key", key).
		Msg("localstore.client.Put")

	bytes, err := json.Marshal(object)
	if err != nil {
		return err
	}

	err = c.store.Put(c.ctx, c.prefix, key, bytes)
	if err != nil {
		return err
	}

	indexers := object.OnUpdate()
	for _, indexer := range indexers {
		log.Ctx(c.ctx).Debug().
			Str("IndexKey", indexer.IndexKey).
			Str("IndexPrefix", indexer.IndexPrefix).
			Msg("localstore.client running update indexer")

		err := c.runIndexer(indexer)
		if err != nil {
			return err
		}
	}

	return nil
}

// Delete will delete the T at the given key. Once the item has been deleted
// the object will be asked for any indexers which can be executed to clean
// up any pointers to this object.
func (c *Client[T]) Delete(key string, object T) error {
	log.Ctx(c.ctx).Debug().
		Str("Key", key).
		Msg("localstore.client.Delete")

	err := c.store.Delete(c.ctx, c.prefix, key)
	if err != nil {
		return err
	}

	indexers := object.OnDelete()
	for _, indexer := range indexers {
		log.Ctx(c.ctx).Debug().
			Str("IndexKey", indexer.IndexKey).
			Str("IndexPrefix", indexer.IndexPrefix).
			Msg("localstore.client running delete indexer")

		err := c.runIndexer(indexer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client[T]) runIndexer(i objectstore.Indexer) error {
	// Uses the indexer to run a transactional update on the prefix
	return c.store.Update(c.ctx, i.IndexPrefix, i.IndexKey, i.Operation)
}

// Only available when using this store as a LocalStore, if using via
// the ObjectStore interface then this will not be available
func (c *Client[T]) GetStore() *LocalStore {
	return c.store
}
