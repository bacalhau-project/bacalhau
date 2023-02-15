//go:build integration && linux

package sqlite

import (
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/localdb/shared"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSQLiteSuite(t *testing.T) {
	testingSuite := new(shared.GenericSQLSuite)
	testingSuite.SetupHandler = func() *shared.GenericSQLDatastore {
		datafile, err := os.CreateTemp("", "sqlite-test-*.db")
		require.NoError(testingSuite.T(), err)
		datastore, err := NewSQLiteDatastore(datafile.Name())
		require.NoError(testingSuite.T(), err)
		return datastore
	}
	suite.Run(t, testingSuite)
}
