package localstore

import (
	"os"
	"path/filepath"
)

type Option func(l *LocalStore)

// WithTestLocation returns an option that will mark the local store as being
// in test mode (so it knows to clean up) and uses a temporary file for
// storage
func WithTestLocation() Option {
	return func(l *LocalStore) {
		dir, _ := os.MkdirTemp("", "bacalhau-objectstore")
		tempFile := filepath.Join(dir, "objectstore.local")
		l.filepath = tempFile
		l.testmode = true
	}
}

// WithLocation returns an option that will set the path to the database file
func WithLocation(filepath string) Option {
	return func(l *LocalStore) {
		l.filepath = filepath
	}
}

// WithPrefixes returns an option that is used to tell the database what
// prefixes we want to have pre-created (if they don't already exist)
func WithPrefixes(prefixes ...string) Option {
	return func(l *LocalStore) {
		l.prefixes = prefixes
	}
}
