package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore/commands"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/exp/slices"
)

const (
	DefaultPermissions = 0666
)

var defaultPrefixes = []string{"job"}

type LocalObjectConfig struct {
	Path          string
	Prefixes      []string
	CallbackHooks commands.CallbackHooks
}

func (l *LocalObjectConfig) Load(options ...Option) {
	for _, opt := range options {
		opt(l)
	}
}

type LocalObjectStore struct {
	path      string
	prefixes  []string
	callbacks *commands.CallbackHooks
	cm        *system.CleanupManager
	database  *bolt.DB
	closed    bool
}

func New(options ...Option) (*LocalObjectStore, error) {
	var err error

	config := &LocalObjectConfig{
		Path:     "",
		Prefixes: defaultPrefixes,
	}
	config.Load(options...)

	store := &LocalObjectStore{
		path:      config.Path,
		prefixes:  []string{},
		callbacks: commands.NewCallbackHooks(),
		cm:        system.NewCleanupManager(),
		closed:    false,
	}

	if store.path == "" {
		// Create and use a temporary file if no useful
		// path was supplied
		dir, _ := os.MkdirTemp("", "bacalhau-objectstore")
		tempFile := filepath.Join(dir, "objectstore.local")
		store.path = tempFile
		store.cm.RegisterCallback(func() error {
			return os.RemoveAll(dir)
		})
	}

	store.database, err = bolt.Open(store.path, DefaultPermissions, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}

	// Pre-create any prefixes that we want to allow up front
	err = store.database.Update(func(tx *bolt.Tx) error {
		for _, prefix := range config.Prefixes {
			_, err := tx.CreateBucketIfNotExists([]byte(prefix))
			if err != nil {
				return ErrInvalidPrefixName(prefix)
			}
			store.prefixes = append(store.prefixes, prefix)
		}
		return nil
	})

	return store, err
}

func (l *LocalObjectStore) CallbackHooks() *commands.CallbackHooks {
	return l.callbacks
}

func (l *LocalObjectStore) Get(ctx context.Context, prefix string, key string) ([]byte, error) {
	if l.closed {
		return nil, ErrDatabaseClosed
	}

	if !slices.Contains[string](l.prefixes, prefix) {
		return nil, ErrNoSuchPrefix(prefix)
	}

	var bytesValue []byte
	err := l.database.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prefix))
		bytesValue = bucket.Get([]byte(key))
		return nil
	})

	return bytesValue, err
}

func (l *LocalObjectStore) Put(ctx context.Context, prefix string, key string, object any) error {
	if l.closed {
		return ErrDatabaseClosed
	}

	if !slices.Contains[string](l.prefixes, prefix) {
		return ErrNoSuchPrefix(prefix)
	}

	var data []byte
	var err error

	// If we were given a byte array, then we will just use that directly,
	// otherwise we'll marshal the object into json bytes
	b, isBytes := object.([]byte)
	if isBytes {
		data = b
	} else {
		data, err = json.Marshal(object)
		if err != nil {
			return err
		}
	}

	err = l.database.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prefix))
		return bucket.Put([]byte(key), data)
	})
	if err != nil {
		return err
	}

	// Check if there are any update callbacks registered for this
	// object type, and if so we want to
	if commands, err := l.callbacks.TriggerUpdate(object); err == nil {
		// err raised from trigger update above refers to whether the object has callbacks
		// or not and so can be ignored if present.

		for _, command := range commands {
			// We want to get the existing data provided by the details in
			// command, and then pass them to command.ModifyFunc. Whatever
			// modify func returns is what we will set the new value to.
			err = l.database.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte(command.Prefix))
				if bucket == nil {
					return ErrNoSuchPrefix(command.Prefix)
				}
				bytesValue := bucket.Get([]byte(command.Key))

				bytes, err := command.Modify(bytesValue)
				if err != nil {
					return err
				}

				return bucket.Put([]byte(command.Key), bytes)
			})

			if err != nil {
				log.Ctx(ctx).Error().
					Str("Prefix", command.Prefix).
					Str("Key", command.Key).
					Err(err).
					Msg("failed to update objectstore from command")
			}
		}
	}

	return nil
}

func (l *LocalObjectStore) Close(ctx context.Context) {
	// Will block until all current transactions complete
	l.closed = true
	l.database.Close()
	l.cm.Cleanup(ctx)
}
