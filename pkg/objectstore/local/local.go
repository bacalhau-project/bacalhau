package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore/index"
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
	Filepath      string
	Prefixes      []string
	CallbackHooks index.CallbackHooks
}

func (l *LocalObjectConfig) Load(options ...Option) {
	for _, opt := range options {
		opt(l)
	}
}

type LocalObjectStore struct {
	path      string
	prefixes  []string
	callbacks *index.CallbackHooks
	cm        *system.CleanupManager
	database  *bolt.DB
	closed    bool
}

func New(ctx context.Context, options ...Option) (*LocalObjectStore, error) {
	var err error

	config := &LocalObjectConfig{
		Filepath: "",
		Prefixes: defaultPrefixes,
	}
	config.Load(options...)

	store := &LocalObjectStore{
		path:      config.Filepath,
		prefixes:  []string{},
		callbacks: index.NewCallbackHooks(),
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
	} else {
		// Ensure the directory for the file exists
		directory := filepath.Dir(store.path)
		err := os.MkdirAll(directory, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	log.Ctx(ctx).Debug().Str("Path", store.path).Msg("opening local database")

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

func (l *LocalObjectStore) CallbackHooks() *index.CallbackHooks {
	return l.callbacks
}

func (l *LocalObjectStore) GetBatch(ctx context.Context, prefix string, keys []string, objects any) error {
	if l.closed {
		return ErrDatabaseClosed
	}

	if !slices.Contains[string](l.prefixes, prefix) {
		return ErrNoSuchPrefix(prefix)
	}

	added := 0
	var buffer []byte

	err := l.database.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prefix))

		first := true
		buffer = append(buffer, '[')

		for _, key := range keys {
			if first {
				first = false
			} else {
				buffer = append(buffer, ',')
			}

			b := bucket.Get([]byte(key))
			buffer = append(buffer, b...)
			added = added + 1
		}
		buffer = append(buffer, ']')

		return nil
	})

	if err != nil {
		return err
	}

	if added != len(keys) {
		return ErrNoSuchKey(strings.Join(keys, ","))
	}

	return json.Unmarshal(buffer, &objects)
}

func (l *LocalObjectStore) Get(ctx context.Context, prefix string, key string, object any) error {
	if l.closed {
		return ErrDatabaseClosed
	}

	if !slices.Contains[string](l.prefixes, prefix) {
		return ErrNoSuchPrefix(prefix)
	}

	var bytesValue []byte
	err := l.database.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prefix))
		bytesValue = bucket.Get([]byte(key))

		if bytesValue == nil {
			return ErrNoSuchKey(key)
		}

		return json.Unmarshal(bytesValue, &object)
	})

	return err
}

func (l *LocalObjectStore) Delete(ctx context.Context, prefix string, key string, object any) error {
	if l.closed {
		return ErrDatabaseClosed
	}

	if !slices.Contains[string](l.prefixes, prefix) {
		return ErrNoSuchPrefix(prefix)
	}

	err := l.database.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(prefix))
		return bucket.Delete([]byte(key))
	})
	if err != nil {
		return err
	}

	// TODO: Triggers should be done via prefix and not type
	if commands, err := l.callbacks.TriggerDelete(prefix, object); err == nil {
		// err raised from trigger update above refers to whether the object has callbacks
		// or not and so can be ignored if present.
		for _, command := range commands {
			err := l.runCallback(command)
			if err != nil {
				log.Ctx(ctx).Error().
					Str("Prefix", command.Prefix).
					Str("Key", command.Key).
					Err(err).
					Msg("failed to delete record from post-delete command")
			}
		}
	}

	return nil
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
	if commands, err := l.callbacks.TriggerUpdate(prefix, object); err == nil {
		// err raised from trigger update above refers to whether the object has callbacks
		// or not and so can be ignored if present.

		for _, command := range commands {
			err := l.runCallback(command)
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

func (l *LocalObjectStore) runCallback(cmd index.IndexCommand) error {
	// We want to get the existing data for the index provided by the details in
	// the provided command. These bytes are passed to the relevant indexing
	// function. Whatever that func returns is what we will set the new value to.
	return l.database.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(cmd.Prefix))
		if bucket == nil {
			return ErrNoSuchPrefix(cmd.Prefix)
		}
		bytesValue := bucket.Get([]byte(cmd.Key))

		bytes, err := cmd.Modify(bytesValue)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(cmd.Key), bytes)
	})
}

func (l *LocalObjectStore) Close(ctx context.Context) error {
	log.Ctx(ctx).Debug().Str("Path", l.path).Msg("closing database")

	// Will block until all current transactions complete
	l.closed = true
	l.database.Close()
	l.cm.Cleanup(ctx)

	return nil
}
