//go:build integration

package postgres

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/localdb/shared"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//	docker run -d \
//	  --name some-postgres \
//	  -p 5432:5432 \
//	  -e POSTGRES_DB=postgres \
//	  -e POSTGRES_USER=postgres \
//	  -e POSTGRES_PASSWORD=postgres \
//	  postgres

func TestPostgresSuite(t *testing.T) {
	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	var datastore *shared.GenericSQLDatastore
	testingSuite := new(shared.GenericSQLSuite)
	testingSuite.SetupHandler = func() *shared.GenericSQLDatastore {
		if runtime.GOOS != "linux" {
			return nil
		}
		if datastore == nil {
			system.Shellout(fmt.Sprintf("docker run -d --name postgres%d -p %d:5432 -e POSTGRES_DB=postgres -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres postgres", port, port))
			for {
				datastore, err = NewPostgresDatastore(
					"localhost",
					port,
					"postgres",
					"postgres",
					"postgres",
					true,
				)
				if err != nil {
					time.Sleep(1 * time.Second)
				} else {
					break
				}
			}
		} else {
			err := datastore.MigrateDown()
			require.NoError(t, err)
			err = datastore.MigrateUp()
			require.NoError(t, err)
		}
		return datastore
	}
	testingSuite.TeardownHandler = func() {
		if runtime.GOOS != "linux" {
			return
		}
		system.Shellout(fmt.Sprintf("docker rm -f postgres%d || true", port))
	}
	suite.Run(t, testingSuite)
}
