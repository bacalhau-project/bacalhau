//go:build unit || !integration

package badgerdb

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/test"
)

func TestStore(t *testing.T) {
	test.RunStoreSuite(t, func(ctx context.Context, dbPath string) (store.ExecutionStore, error) {
		return NewStore(dbPath)
	})
}
