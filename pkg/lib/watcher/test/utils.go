package test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
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
		_ = db.Close()
	})

	return db
}

// CreateSerializer creates a new JSONSerializer for testing
func CreateSerializer(t *testing.T) *watcher.JSONSerializer {
	t.Helper()

	serializer := watcher.NewJSONSerializer()
	require.NoError(t, serializer.RegisterType("TestObject", reflect.TypeOf(TestObject{})))

	return serializer
}
