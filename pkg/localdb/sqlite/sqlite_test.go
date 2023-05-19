//go:build integration || !unit

package sqlite

import (
	"os"
	"runtime"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/localdb/shared"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSQLiteSuite(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("Test only runs on linux and not %s", runtime.GOOS)
	}

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
