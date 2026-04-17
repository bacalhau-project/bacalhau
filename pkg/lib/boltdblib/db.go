package boltdblib

import (
	"errors"
	"time"

	bolt "go.etcd.io/bbolt"
	bbolterrors "go.etcd.io/bbolt/errors"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

const component = "BoltDB"
const defaultDatabasePermissions = 0600

func Open(path string) (*bolt.DB, error) {
	database, err := bolt.Open(path, defaultDatabasePermissions, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		if errors.Is(err, bbolterrors.ErrTimeout) {
			return nil, newBoltDBInUseError(path)
		}
		return nil, bacerrors.Wrapf(err, "failed to open database at %s", path)
	}
	return database, nil
}

func newBoltDBInUseError(path string) bacerrors.Error {
	return bacerrors.Newf("db is in use: %s", path).
		WithHint(`most likely another bacalhau is running and using the same data directory. To resolve, either:
1. Ensure that no other bacalhau process is running
2. Select a different data directory using the '--data-dir <new_path>' or '-c %s=<new_path>' flag with your serve command
`, types.DataDirKey).
		WithCode(bacerrors.ConfigurationError).
		WithComponent(component)
}
