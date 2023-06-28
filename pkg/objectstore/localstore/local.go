package localstore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	bolt "go.etcd.io/bbolt"
)

const (
	DefaultDatabasePermissions = 0666
	DefaultKeyListCapacity     = 32
)

type LocalStore struct {
	database *bolt.DB
	filepath string
	testmode bool
	prefixes []string
}

// NewLocalStore creates a new localstore and returns a pointer to it, or
// an error if there is a problem during its construction.  Once the store
// has been configured any prefixes provided to this function will be created
// if they do not already exist.
func NewLocalStore(ctx context.Context, options ...Option) (*LocalStore, error) {
	var err error

	store := &LocalStore{}
	for _, opt := range options {
		opt(store)
	}

	if store.filepath == "" {
		return nil, errors.New("no filepath option was provided and it is necessary")
	}

	if len(store.prefixes) == 0 {
		return nil, errors.New("no prefixes were provided ahead of time for the database")
	}

	log.Ctx(ctx).Info().
		Str("File", store.filepath).
		Str("Prefixes", strings.Join(store.prefixes, ",")).
		Msg("opening localstore database")

	store.database, err = bolt.Open(store.filepath, DefaultDatabasePermissions, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}

	err = store.database.Update(func(tx *bolt.Tx) error {
		errs := lo.Map(store.prefixes, func(prefix string, index int) error {
			_, err := tx.CreateBucketIfNotExists([]byte(prefix))
			return err
		})

		err, foundErr := lo.Find(errs, func(e error) bool { return e != nil })
		if foundErr {
			return err
		}

		return nil
	})

	return store, err
}

// Get retrieves any data stored in the database with the specified prefix
// and key.  You can think of the prefix as a bucket, a namespace, or other
// named container which can group together IDs of a given type (to add an
// extra level of uniqueness
func (s *LocalStore) Get(ctx context.Context, prefix string, key string) ([]byte, error) {
	var bytesValue []byte

	if !s.verifyPrefix(prefix) {
		return nil, objectstore.NewErrInvalidPrefix(prefix)
	}

	log.Ctx(ctx).Debug().
		Str("Prefix", prefix).
		Str("Key", key).
		Msg("database.Get()")

	err := s.database.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prefix))
		bytesValue = bucket.Get([]byte(key))

		if bytesValue == nil {
			return objectstore.NewErrNotFound(key)
		}

		return nil
	})

	return bytesValue, err
}

// GetBatch retrieves one record for each key provided
func (s *LocalStore) GetBatch(ctx context.Context, prefix string, keys []string) (map[string][]byte, error) {
	if !s.verifyPrefix(prefix) {
		return nil, objectstore.NewErrInvalidPrefix(prefix)
	}

	log.Ctx(ctx).Debug().
		Str("Prefix", prefix).
		Str("Key", strings.Join(keys, ",")).
		Msg("database.GetBatch()")

	results := make(map[string][]byte)
	err := s.database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(prefix))

		for _, k := range keys {
			data := b.Get([]byte(k))
			if data != nil {
				results[k] = data
			}
		}

		return nil
	})

	return results, err
}

// List returns a list of keys found within the specified prefix
func (s *LocalStore) List(ctx context.Context, prefix string, keyPrefix string) ([]string, error) {
	if !s.verifyPrefix(prefix) {
		return nil, objectstore.NewErrInvalidPrefix(prefix)
	}

	log.Ctx(ctx).Debug().
		Str("Prefix", prefix).
		Str("KeyPrefix", keyPrefix).
		Msg("database.List()")

	// Assume we've got a few
	keys := make([]string, DefaultKeyListCapacity)
	err := s.database.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(prefix)).Cursor()

		p := []byte(keyPrefix)
		for k, _ := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, _ = c.Next() {
			keys = append(keys, string(k))
		}

		return nil
	})

	return keys, err
}

func (s *LocalStore) Update(ctx context.Context, prefix string, key string, op func([]byte) ([]byte, error)) error {
	if !s.verifyPrefix(prefix) {
		return objectstore.NewErrInvalidPrefix(prefix)
	}

	log.Ctx(ctx).Debug().
		Str("Prefix", prefix).
		Str("Key", key).
		Msg("database.Update()")

	k := []byte(key)
	return s.database.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prefix))
		data := bucket.Get(k)
		bytes, err := op(data)
		if err != nil {
			return err
		}

		return bucket.Put(k, bytes)
	})
}

// Delete removes the value for the specified key from the provided
// prefix, and if any delete callbacks were registered, will also run
// those before returning
func (s *LocalStore) Delete(ctx context.Context, prefix string, key string) error {
	if !s.verifyPrefix(prefix) {
		return objectstore.NewErrInvalidPrefix(prefix)
	}

	log.Ctx(ctx).Debug().
		Str("Prefix", prefix).
		Str("Key", key).
		Msg("database.Delete()")

	err := s.database.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prefix))
		return bucket.Delete([]byte(key))
	})

	return err
}

// Put will store `data` in the prefix namespace/bucket/container with the
// provided key.  If the Put fails then an error is returned, otherwise
// nil.
func (s *LocalStore) Put(ctx context.Context, prefix string, key string, value []byte) error {
	if !s.verifyPrefix(prefix) {
		return objectstore.NewErrInvalidPrefix(prefix)
	}

	log.Ctx(ctx).Debug().
		Str("Prefix", prefix).
		Str("Key", key).
		Msg("database.Put()")

	err := s.database.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prefix))
		return bucket.Put([]byte(key), value)
	})
	if err != nil {
		return err
	}

	return nil
}

// Close will close the database, after which it should not be usable
func (s *LocalStore) Close(ctx context.Context) error {
	err := s.database.Close()

	log.Ctx(ctx).Info().Msg("closing localstore database")

	if s.testmode {
		os.Remove(s.filepath)
	}

	return err
}

func (s *LocalStore) verifyPrefix(prefix string) bool {
	return lo.Contains(s.prefixes, prefix)
}
