package sqlite

import (
	"fmt"

	"database/sql"

	"github.com/filecoin-project/bacalhau/pkg/localdb/shared"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "modernc.org/sqlite"
)

func NewSQLiteDatastore(filename string) (*shared.GenericSQLDatastore, error) {
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		return nil, err
	}
	datastore, err := shared.NewGenericSQLDatastore(
		db,
		"sqlite",
		fmt.Sprintf("sqlite://%s", filename),
	)
	if err != nil {
		return nil, err
	}
	err = datastore.MigrateUp()
	if err != nil {
		return nil, err
	}
	return datastore, err
}
