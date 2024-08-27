//go:build unit || !integration

package boltdb

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/test"
)

func TestStore(t *testing.T) {
	test.RunStoreSuite(t, func(ctx context.Context, dbPath string) (store.ExecutionStore, error) {
		dbFile := filepath.Join(dbPath, "test.boltdb")
		return NewStore(ctx, dbFile)
	})
}
