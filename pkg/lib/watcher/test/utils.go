package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

// CreateBoltDB creates a new BoltDB database for testing
func CreateBoltDB(t *testing.T) *bbolt.DB {
	t.Helper()

	// Create a temporary file for the test database
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.db")

	// Open the database
	db, err := bbolt.Open(tempFile, os.FileMode(os.O_RDWR), nil)
	require.NoError(t, err)

	// Register cleanup to close the database after the test completes
	t.Cleanup(func() {
		db.Close()
	})

	return db
}
